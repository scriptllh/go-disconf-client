/**
 * @Author: llh
 * @Date:   2018-01-25 16:08:29
 * @Last Modified by:   llh
 */

package disconf_client

import (
	"strings"
	"fmt"
	"sync"
	"github.com/sirupsen/logrus"
	"github.com/samuel/go-zookeeper/zk"
	"encoding/json"
)

type Client struct {
	retryTimes        int
	retrySleepSeconds int
	downloadDir       string
	serverHost        string
	app               string
	version           string
	enableRemote      bool
	env               string
	ignore            string
	debug             bool
	fetcher           IFetcher
	watch             IWatch
	store             *Store
}

type ClientOption func(*Client)

func WithRetryTimes(retryTimes int) ClientOption {
	return func(c *Client) {
		c.retryTimes = retryTimes
	}
}

func WithRetrySleepSeconds(retrySleepSeconds int) ClientOption {
	return func(c *Client) {
		c.retrySleepSeconds = retrySleepSeconds
	}
}

func WithDownloadDir(downloadDir string) ClientOption {
	return func(c *Client) {
		c.downloadDir = downloadDir
	}
}

func WithIgnore(ignore string) ClientOption {
	return func(c *Client) {
		c.ignore = ignore
	}
}

func NewConf(serverHost, app, version, env string, enableRemote, debug bool, conf interface{}, opts ... ClientOption) error {
	defaultClient := &Client{
		retryTimes:        RETRY_TIMES,
		retrySleepSeconds: RETRY_SLEEP_SECONDS,
		downloadDir:       DEFAULT_DOWNLOAD_DIR,
		ignore:            EMPTY_STRING,
	}
	for _, o := range opts {
		o(defaultClient)
	}
	var fetcher IFetcher
	var watch IWatch
	fetcher = &Fetcher{
		retryTime:         defaultClient.retryTimes,
		retrySleepSeconds: defaultClient.retrySleepSeconds,
		downloadDir:       defaultClient.downloadDir,
		hostList:          strings.Split(serverHost, COMMA_SPLIT),
	}
	zkHosts, errs := fetcher.getZkHost()
	if len(errs) > 0 {
		return fmt.Errorf("get zk hosts [errs:%v]", errs)
	}
	watch, err := newWatch(zkHosts, app, version, env, debug)
	if err != nil {
		return err
	}
	client := &Client{
		retryTimes:        defaultClient.retryTimes,
		retrySleepSeconds: defaultClient.retrySleepSeconds,
		downloadDir:       defaultClient.downloadDir,
		serverHost:        serverHost,
		app:               app,
		version:           version,
		enableRemote:      enableRemote,
		env:               env,
		ignore:            defaultClient.ignore,
		debug:             debug,
		fetcher:           fetcher,
		store:             &Store{conf},
		watch:             watch,
	}
	if err := client.initConf(); err != nil {
		return err
	}
	return nil
}

const (
	RETRY_TIMES          = 3
	RETRY_SLEEP_SECONDS  = 5
	DEFAULT_DOWNLOAD_DIR = "./disconf/download/"
	COMMA_SPLIT          = ","
	SUFFIX_PREFIX_URL    = "?app=%v&env=%v&version=%v"
	SUFFIX_KEY           = "&key=%v"
)

func (c *Client) suffixPrefixUrlString() string {
	return fmt.Sprintf(SUFFIX_PREFIX_URL, c.app, c.env, c.version)
}

func (c *Client) suffixKeyString(key string) string {
	return fmt.Sprintf(SUFFIX_KEY, key)
}

func (c *Client) initConf() error {
	if !c.enableRemote {
		if err := c.store.loadPropertiesDir(c.downloadDir, c.ignore); err != nil {
			return err
		}
		return nil
	}
	confs, errs := c.fetcher.getAllConf(c.suffixPrefixUrlString())
	if len(errs) > 0 {
		return fmt.Errorf("get all conf from server [errs:%v]", errs)
	}
	if err := c.downloadFiles(confs); err != nil {
		return err
	}
	if err := c.store.loadConf(confs, c.downloadDir, c.ignore); err != nil {
		return err
	}
	go c.autoLoad(confs)
	return nil
}

func (c *Client) autoLoad(confs []*Result) {
	respChan := make(chan watchResponse, 16)
	localHostPath, err := c.watch.getLocalHostPath()
	if err != nil {
		logrus.Errorf("get local hosts path [err:%v]", err)
	}
	for _, conf := range confs {
		if ContainString(c.ignore, conf.Name) {
			continue
		}
		if (conf.Genre == DISCONF_TYPE_FILE && strings.HasSuffix(conf.Name, FILE_PROPERTIES)) || conf.Genre == DISCONF_TYPE_ITEM {
			if err := c.watch.createZkDir(conf.Genre, conf.Name); err != nil {
				logrus.Errorf("create file or item zk dir [err:%v]", err)
			}
			monitorPath, err := c.watch.getBaseUrl(conf.Name, conf.Genre)
			if err != nil {
				logrus.Errorf("get zk base path [err:%v]", err)

			}
			var byteValue []byte
			if conf.Genre == DISCONF_TYPE_FILE {
				fileMap, err := c.store.loadProperties(c.downloadDir, conf.Name, INIT_CONF)
				byteValue, err = json.Marshal(fileMap)
				//fmt.Println(string(byteValue))
				if err != nil {
					logrus.Errorf("marshal value [err:%v]", err)
				}
			} else {
				byteValue = []byte(conf.Value)
			}
			if err := c.watch.createZkPath(monitorPath+localHostPath, zk.FlagEphemeral, byteValue); err != nil {
				logrus.Errorf("create zk temp path [err:%v]", err)
			}
			go c.watch.watchPath(conf.Name, conf.Genre, respChan)
		}
	}
	for {
		select {
		case resp := <-respChan:
			byteValue,err := c.autoLoadProperties(resp)
			if err != nil {
				logrus.Errorf("auto load properties [key:%v] [err:%v]", resp.key, err)
			}
			monitorPath, err := c.watch.getBaseUrl(resp.key, resp.disconfType)
			if err != nil {
				logrus.Errorf("get zk base path [err:%v]", err)
			}
			if err := c.watch.setZkValue(monitorPath+localHostPath, byteValue); err != nil {
				logrus.Errorf("create zk temp path [err:%v]", err)
			}
			go c.watch.watchPath(resp.key, resp.disconfType, respChan)
			logrus.Infof("auto load [key:%v]", resp.key)
		}
	}
}

func (c *Client) autoLoadProperties(resp watchResponse) ([]byte, error) {
	if resp.err != nil {
		return nil,fmt.Errorf("watch [key:%v] [err:%v]", resp.key, resp.err)
	}
	if !(resp.disconfType == DISCONF_TYPE_ITEM || resp.disconfType == DISCONF_TYPE_FILE) {
		return nil,fmt.Errorf("disconf type err")
	}
	if resp.disconfType == DISCONF_TYPE_ITEM {
		value, errs := c.fetcher.getValue(c.suffixPrefixUrlString() + c.suffixKeyString(resp.key))
		if len(errs) > 0 {
			return nil,fmt.Errorf("get value [key:%v] [errs:%v]", resp.key, errs)
		}
		if err := c.store.loadItem(resp.key, value, AUTO_CONF); err != nil {
			return nil,err
		}
		return []byte(value),nil
	}
	if errs := c.fetcher.downloadFile(c.suffixPrefixUrlString()+c.suffixKeyString(resp.key), resp.key); len(errs) > 0 {
		return nil,fmt.Errorf("download file [fileName:%v] [errs:%v]", resp.key, errs)
	}
	fileMap, err := c.store.loadProperties(c.downloadDir, resp.key, AUTO_CONF)
	if err != nil {
		return nil,fmt.Errorf("load file properties [fileName:%v] [errs:%v]", resp.key, err)
	}
	byteValue, err := json.Marshal(fileMap)
	if err != nil {
		logrus.Errorf("marshal value [err:%v]", err)
	}

	return byteValue,nil
}

func (c *Client) downloadFiles(confs []*Result) error {
	errs := []error{}
	wg := &sync.WaitGroup{}
	var mutex sync.Mutex
	for _, conf := range confs {
		if conf.Genre == DISCONF_TYPE_FILE {
			wg.Add(1)
			go func(fileName string) {
				defer wg.Done()
				if fErrs := c.fetcher.downloadFile(c.suffixPrefixUrlString()+c.suffixKeyString(fileName), fileName); len(fErrs) > 0 {
					mutex.Lock()
					errs = append(errs, fErrs...)
					mutex.Unlock()
				}
			}(conf.Name)
		}
	}
	wg.Wait()
	if len(errs) > 0 {
		fmt.Errorf("download file from server [errs:%v]", errs)
	}
	return nil
}

