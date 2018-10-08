#### go disconf 客户端
***
1.使用说明
   
 
  * 传一个结构体的指针,支持数据类型(支持int、int64、string、bool、float32、float64)
 
  * 支持两种tag:conf、auto
  
  * 支持默认参数（WithRetryTimes(3)、WithRetrySleepSeconds(5)、WithDownloadDir(./disconf/download/)、WithIgnore）
  * tag conf 是属性文件中的名称，如果加了auto:"true"表示该属性在disconf服务端更新之后，客户端会自动加载
  * example
  
```
  type Conf struct {
	UserName string `conf:"mysql.username"`
	Password string `conf:"mysql.password" auto:"true"`
	A        int    `conf:"a" auto:"true"`
	TextGBK  string `conf:"textGBK" auto:"true"`
}
```

```
conf := &Conf{UserName: "root", Password: "dsdhjhj"}
	if err := NewConf(
		"127.0.0.1",
		"disconf_demo",
		"222",
		"dev",
		true,
		false,
		conf,
		WithDownloadDir("./disconf/download/")); err != nil {
		t.Fatalf("new conf [err:%v]", err)
	}
	for {
	       fmt.Println("a", conf.Password)
	       time.Sleep(5 * time.Second)
		}
```


***

###### 整体架构

   ![avatar](https://github.com/scriptllh/go-disconf-client/blob/dev/docs/flow.svg)

  
***
  
##### 特性
   *  支持自定义配置文件下载路径
    * 支持配置文件和配置项
    * 支持可配置的只加载本地配置
    * 不需要重启更改配置文件或配置项
    *  应用程序无感知



