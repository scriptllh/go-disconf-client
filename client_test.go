/**
 * @Author: llh
 * @Date:   2018-01-25 16:08:29
 * @Last Modified by:   llh
 */

package disconf_client

import (
	"testing"
	"time"
	"fmt"
)

type Conf struct {
	UserName string `conf:"mysql.username"`
	Password string `conf:"mysql.password" auto:"true"`
	A        int    `conf:"a" auto:"true"`
	TextGBK  string `conf:"textGBK" auto:"true"`
}

func TestNewConf(t *testing.T) {
	conf := &Conf{UserName: "2", Password: "d"}
	if err := NewConf(
		"http://127.0.0.1",
		"disconf_demo",
		"1_0_0_0",
		"dev",
		true,
		false,
		conf); err != nil {
		t.Fatalf("new conf [err:%v]", err)
	}
	for {
		fmt.Println("a", conf.Password)
		fmt.Println(conf.TextGBK)
		time.Sleep(5 * time.Second)
	}
}
