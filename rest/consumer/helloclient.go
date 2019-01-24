package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/chinx/helloworld/rest/common/config"
	"github.com/chinx/helloworld/rest/common/servicecenter/v3"
)

var caches = &sync.Map{}

func main() {
	// 配置文件加载
	err := config.LoadConfig("./conf/microservice.yaml")
	if err != nil {
		log.Fatalf("load config file faild: %s", err)
	}

	// 注册自身微服务
	svcID := registerService()

	// 服务发现 provider 实例信息
	discoveryProviderAndCache(svcID)

	// 与 provider 通讯
	log.Println(sayHello())

	// 提供对外服务，将请求转发到 helloServer 处理，验证 watch 功能
	sayHelloServer(svcID)
}

// 提供对外服务，将请求转发到 helloServer 处理，验证 watch 功能
func sayHelloServer(svcID string)  {
	// 启动 provider 订阅
	go watch(svcID)

	// 启动 http 监听
	http.HandleFunc("/sayhello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sayHello()))
	})
	err := http.ListenAndServe(":8090", nil)
	log.Println(err)
}

// 注册微服务信息
func registerService() string {
	// 微服务未注册则注册其信息
	cli := v3.NewClient(config.Registry.Address, config.Tenant.Domain)
	svcID, _ := cli.GetServiceID(config.Service)
	if svcID == "" {
		var err error
		svcID, err = cli.RegisterService(config.Service)
		if err != nil {
			log.Fatalln(err)
		}
	}
	return svcID
}

// 服务发现 provider  实例信息, 并缓存
func discoveryProviderAndCache(svcID string) {
	// 服务发现 provider 实例信息
	cli := v3.NewClient(config.Registry.Address, config.Tenant.Domain)
	pris, err := cli.Discovery(svcID, config.Provider)
	if err != nil {
		log.Fatalln(err)
	}

	if len(pris) == 0 {
		log.Fatalf("provider not found, serviceName: %s appID: %s, version: %s",
			config.Provider.Name, config.Provider.AppID, config.Provider.Version)
	}

	if len(pris[0].Endpoints) == 0 {
		log.Fatalln("provider endpoints is empty")
	}

	// 缓存 provider 实例信息
	caches.Store(config.Provider, pris)
}

// 订阅 provider 变更
func watch(svcID string) {
	cli := v3.NewClient(config.Registry.Address, config.Tenant.Domain)
	err := cli.WatchService(svcID, watchCallback)
	if err != nil {
		log.Println(err)
	}
}

// 订阅回调，更新 provider 缓存
func watchCallback(data *proto.WatchInstanceResponse) {
	log.Println("reply from watch service")
	prisCache, ok := caches.Load(config.Provider)
	if !ok {
		log.Printf("provider \"%s\" not found", config.Provider.Name)
		return
	}
	pris := prisCache.([]*proto.MicroServiceInstance)
	renew := false
	for i := 0; i < len(pris); i ++{
		if pris[i].InstanceId == data.Instance.InstanceId{
			pris[i] = data.Instance
			renew = true
			break
		}
	}
	if !renew {
		pris = append(pris, data.Instance)
	}
	caches.Store(config.Provider, pris)
}

// 获取在线的 provider endpoint
func getProviderEndpoint() (string, error) {
	prisCache, ok := caches.Load(config.Provider)
	if !ok {
		return "", fmt.Errorf("provider \"%s\" not found", config.Provider.Name)

	}

	endpoint := ""
	pris := prisCache.([]*proto.MicroServiceInstance)

	for i := 0; i < len(pris); i ++{
		if pris[i].Status == "UP"{
			endpoint = pris[i].Endpoints[0]
			break
		}
	}

	if endpoint != ""{
		addr, err := url.Parse(endpoint)
		if err != nil {
			return "", fmt.Errorf("parse provider endpoint faild: %s", err)
		}
		if addr.Scheme == "rest" {
			addr.Scheme = "http"
		}
		return addr.String(), nil
	}
	return "", fmt.Errorf("provider \"%s\" endpoint not found", config.Provider.Name)
}

// 与 provider 通讯
func sayHello() string {
	addr, err := getProviderEndpoint()
	if err != nil {
		return err.Error()
	}
	req, err := http.NewRequest(http.MethodGet, addr+"/hello", nil)
	if err != nil {
		return fmt.Sprintf("create request faild: %s", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("do request faild: %s", err)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("read response body faild: %s, body: %s", err, string(data))
	}

	log.Printf("reply form provider: %s", string(data))
	return string(data)
}
