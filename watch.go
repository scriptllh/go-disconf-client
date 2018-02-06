/**
 * @Author: llh
 * @Date:   2018-01-25 16:08:29
 * @Last Modified by:   llh
 */

package disconf_client

import (
	"github.com/samuel/go-zookeeper/zk"
	"fmt"
	"time"
	"strings"
	"net"
	"crypto/md5"
	"encoding/hex"
	"io"
	"encoding/base64"
	"crypto/rand"
)

type IWatch interface {
	initZk() error

	watchPath(key string, disconfType int, respChan chan watchResponse)

	createZkPath(path string, zkFlag int32, value []byte) error

	getBaseUrl(key string, disconfType int) (string, error)

	createZkDir(disconfType int, key string) error

	getLocalHostPath() (string, error)

	setZkValue(path string, value []byte) error
}

type Watch struct {
	servers      []string
	zKClientConn *zk.Conn
	app          string
	version      string
	env          string
	debug        bool
}

const (
	RE_CONNECT_TIMES = 3
	ZK_TIMEOUT       = 5
	GO_TIMEOUT       = 3
	PORT             = "8080"
)

type watchResponse struct {
	err         error
	disconfType int
	key         string
}

func newWatch(serverStr string, app, version, env string, debug bool) (*Watch, error) {
	servers := strings.Split(serverStr, COMMA_SPLIT)
	watch := &Watch{
		servers: servers,
		app:     app,
		version: version,
		env:     env,
		debug:   debug,
	}
	if err := watch.initZk(); err != nil {
		if debug {
			return nil, err
		}
		ok := false
		for i := 0; i < RE_CONNECT_TIMES; i++ {
			if err = watch.initZk(); err == nil {
				ok = true
				break
			}
		}
		if !ok {
			return nil, err
		}
	}
	return watch, nil
}

func (w *Watch) initZk() error {
	if !w.isConnected() {
		conn, connChan, err := zk.Connect(w.servers, time.Duration(ZK_TIMEOUT*time.Second))
		if err != nil {
			return err
		}
		for {
			isConnected := false
			select {
			case connEvent := <-connChan:
				if connEvent.State == zk.StateConnected {
					isConnected = true
				}
			case _ = <-time.After(time.Second * GO_TIMEOUT):
				return fmt.Errorf("connect to zookeeper server timeout!")
			}
			if isConnected {
				break
			}
		}
		w.zKClientConn = conn
	}
	return nil
}

func (w *Watch) isConnected() bool {
	if w.zKClientConn == nil || w.zKClientConn.State() != zk.StateConnected {
		return false
	}
	return true
}

func (w *Watch) watchPath(key string, disconfType int, respChan chan watchResponse) {
	monitorPath, err := w.getBaseUrl(key, disconfType)
	if err != nil {
		respChan <- watchResponse{err, disconfType, key}
		return
	}
	_, _, keyEventCh, err := w.zKClientConn.GetW(monitorPath)
	if err != nil {
		respChan <- watchResponse{err, disconfType, key}
		return
	}
	for {
		select {
		case e := <-keyEventCh:
			if e.Type == zk.EventNodeDataChanged {
				respChan <- watchResponse{e.Err, disconfType, key}
				return
			}
		}
	}
}

func (w *Watch) getBaseUrl(key string, disconfType int) (string, error) {
	if !(disconfType == DISCONF_TYPE_FILE || disconfType == DISCONF_TYPE_ITEM) {
		return EMPTY_STRING, fmt.Errorf("disconf type err")
	}
	if disconfType == DISCONF_TYPE_FILE {
		return fmt.Sprintf("/disconf/%v_%v_%v/file/%v", w.app, w.version, w.env, key), nil
	}
	return fmt.Sprintf("/disconf/%v_%v_%v/item/%v", w.app, w.version, w.env, key), nil
}

func (w *Watch) getLocalHostPath() (string, error) {
	uuid, err := getGuid()
	if err != nil {
		return EMPTY_STRING, err
	}
	ip, err := getLocalIp()
	if err != nil {
		return EMPTY_STRING, err
	}
	return fmt.Sprintf("/%v_%v_%v", ip, PORT, uuid), nil
}

func (w *Watch) createZkPath(path string, zkFlag int32, value []byte) error {
	isExist, _, err := w.zKClientConn.Exists(path)
	if err != nil {
		return err
	}
	if zkFlag == zk.FlagEphemeral {
		isExist = false
	}
	if !isExist {
		zkPath, err := w.zKClientConn.Create(path, value, zkFlag, zk.WorldACL(zk.PermAll))
		if err != nil {
			return err
		}
		if zkPath != path {
			return err
		}
	}
	return nil
}

func (w *Watch) setZkValue(path string, value []byte) error {
	_, err := w.zKClientConn.Set(path, value,-1)
	if err != nil {
		return err
	}
	return nil
}

func (w *Watch) createZkDir(disconfType int, key string) error {
	ip, err := getLocalIp()
	if err != nil {
		return err
	}
	if err := w.createZkPath("/disconf", 0, nil); err != nil {
		return err
	}
	if err := w.createZkPath(fmt.Sprintf("/disconf/%v_%v_%v", w.app, w.version, w.env), 0, nil); err != nil {
		return err
	}
	if !(disconfType == DISCONF_TYPE_FILE || disconfType == DISCONF_TYPE_ITEM) {
		return fmt.Errorf("disconf type err")
	}
	if disconfType == DISCONF_TYPE_FILE {
		if err := w.createZkPath(fmt.Sprintf("/disconf/%v_%v_%v/file", w.app, w.version, w.env), 0, []byte (ip)); err != nil {
			return err
		}
		if err := w.createZkPath(fmt.Sprintf("/disconf/%v_%v_%v/file/%v", w.app, w.version, w.env, key), 0, []byte("")); err != nil {
			return err
		}
		return nil
	}
	if err := w.createZkPath(fmt.Sprintf("/disconf/%v_%v_%v/item", w.app, w.version, w.env), 0, []byte (ip)); err != nil {
		return err
	}
	if err := w.createZkPath(fmt.Sprintf("/disconf/%v_%v_%v/item/%v", w.app, w.version, w.env, key), 0, []byte("")); err != nil {
		return err
	}
	return nil
}

func getLocalIp() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return EMPTY_STRING, err
	}
	var ip string
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}
	return ip, err
}

func getMd5String(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func getGuid() (string, error) {
	b := make([]byte, 48)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return EMPTY_STRING, err
	}
	return getMd5String(base64.URLEncoding.EncodeToString(b)), nil
}
