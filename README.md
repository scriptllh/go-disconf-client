#### go disconf 客户端
***
1.使用说明
   
 
  * 只能传一个结构体的指针，且该结构体中只能有基础数据类型(支持int、int64、string、bool、float32、float64)
 
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

 



  

  
