/**
 * @Author: llh
 * @Date:   2018-01-25 16:08:29
 * @Last Modified by:   llh
 */

package disconf_client

import (
	"github.com/parnurzeal/gorequest"
	"time"
	"os"
	"io/ioutil"
	"fmt"
	"strings"
	"net/http"
)

type IFetcher interface {
	getValue(suffixUrl string) (string, []error)

	downloadFile(suffixUrl, fileName string) []error

	getAllConf(suffixUrl string) ([]*Result, []error)

	getZkHost() (string, []error)
}
type Fetcher struct {
	//文件下载目录
	downloadDir string

	// 获取远程配置 重试次数
	retryTime int

	// 获取远程配置 重试时休眠时间 (s)
	retrySleepSeconds int

	// host List
	hostList []string
}

type zooHostsResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Value   string `json:"value"`
}

type confListResp struct {
	Success string      `json:"success"`
	Message interface{} `json:"message"`
	Page struct {
		Results []*Result `json:"result"`
	} `json:"page"`
}

type Result struct {
	Id      int    `json:"id"`
	Genre   int    `json:"type"`
	Status  int    `json:"status"`
	Name    string `json:"name"`
	Value   string `json:"value"`
	AppId   int    `json:"appId"`
	Version string `json:"version"`
	EnvId   int    `json:"envId"`
}

const (
	EMPTY_STRING             = ""
	PREFIX_HTTP              = "http://"
	PREFIX_HTTPS             = "https://"
	DISCONF_STORE_ACTION     = "/api/config/list"
	DISCONF_ITEM_ACTION      = "/api/config/item"
	DISCONF_FILE_ACTION      = "/api/config/file"
	DISCONF_ZOO_HOSTS_ACTION = "/api/zoo/hosts"
	STRING_TRUE              = "true"
	ZOO_SUCCESS_STATUS       = 1
)

type itemResp struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Value   string `json:"value"`
}

func (f Fetcher) getValue(suffixUrl string) (string, []error) {
	urls := f.getUrls(DISCONF_ITEM_ACTION + suffixUrl)
	var resp itemResp
	errs := []error{}
	for _, url := range urls {
		_, _, httpErrs := gorequest.New().Get(url).
			Retry(f.retryTime, time.Duration(f.retrySleepSeconds)*time.Second, http.StatusBadRequest, http.StatusInternalServerError).
			EndStruct(&resp)
		if len(httpErrs) <= 0 {
			return resp.Value, nil
		}
		errs = append(errs, httpErrs...)
	}
	return EMPTY_STRING, errs
}

func (f Fetcher) downloadFile(suffixUrl, fileName string) []error {
	errs := []error{}
	_, err := os.Stat(f.downloadDir)
	if err != nil {
		err = os.MkdirAll(f.downloadDir, 0777)
		if err != nil {
			return append(errs, err)
		}
	}
	bodyBytes, errs := f.httpEndByte(DISCONF_FILE_ACTION + suffixUrl)
	if len(errs) > 0 {
		return errs
	}
	if err = ioutil.WriteFile(f.downloadDir+fileName, bodyBytes, os.ModeAppend); err != nil {
		return append(errs, err)
	}
	return nil
}

func (f Fetcher) getAllConf(suffixUrl string) ([]*Result, []error) {
	urls := f.getUrls(DISCONF_STORE_ACTION + suffixUrl)
	var resp confListResp
	errs := []error{}
	for _, url := range urls {
		_, _, httpErrs := gorequest.New().Get(url).
			Retry(f.retryTime, time.Duration(f.retrySleepSeconds)*time.Second, http.StatusBadRequest, http.StatusInternalServerError).
			EndStruct(&resp)
		if len(httpErrs) <= 0 {
			if resp.Success != STRING_TRUE {
				errs = append(errs, fmt.Errorf("get all conf %v", resp.Message))
				continue
			}
			return resp.Page.Results, nil
		}
		errs = append(errs, httpErrs...)
	}
	return nil, errs
}

func (f Fetcher) getZkHost() (string, []error) {
	urls := f.getUrls(DISCONF_ZOO_HOSTS_ACTION)
	var resp zooHostsResp
	errs := []error{}
	for _, url := range urls {
		_, _, httpErrs := gorequest.New().Get(url).
			Retry(f.retryTime, time.Duration(f.retrySleepSeconds)*time.Second, http.StatusBadRequest, http.StatusInternalServerError).
			EndStruct(&resp)
		if len(httpErrs) <= 0 {
			if resp.Status != ZOO_SUCCESS_STATUS {
				errs = append(errs, fmt.Errorf("get zoo hosts [err:%v]", resp.Message))
				continue
			}
			return resp.Value, nil
		}
		errs = append(errs, httpErrs...)
	}
	return EMPTY_STRING, errs
}

func (f Fetcher) httpEndByte(suffixUrl string) ([]byte, []error) {
	errs := []error{}
	urls := f.getUrls(suffixUrl)
	for _, url := range urls {
		_, bodyBytes, httpErrs := gorequest.New().Get(url).
			Retry(f.retryTime, time.Duration(f.retrySleepSeconds)*time.Second, http.StatusBadRequest, http.StatusInternalServerError).
			EndBytes()
		if len(httpErrs) <= 0 {
			return bodyBytes, nil
		}
		errs = append(errs, httpErrs...)
	}
	return nil, errs
}

func (f Fetcher) getUrls(suffixUrl string) []string {
	urls := []string{}
	for _, host := range f.hostList {
		if !strings.HasPrefix(host, PREFIX_HTTP) {
			if !strings.HasPrefix(host, PREFIX_HTTPS) {
				host = PREFIX_HTTP + host
			}
		}
		host += suffixUrl
		urls = append(urls, host)
	}
	return urls
}
