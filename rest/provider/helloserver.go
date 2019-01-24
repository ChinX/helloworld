package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/chinx/helloworld/rest/common/config"
	"github.com/chinx/helloworld/rest/common/servicecenter/v3"
)

var HeartbeatInterval = 30 * time.Second

func main() {
	// 加载配置文件
	err := config.LoadConfig("./conf/microservice.yaml")
	if err != nil {
		log.Fatalf("load config file faild: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	// 注册微服务与实例，启动心跳
	go registerAndHeartbeat(ctx)

	// 启动 http 监听
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello world"))
	})
	err = http.ListenAndServe(config.Instance.ListenAddress, nil)
	log.Println(err)
	cancel()
}

func registerAndHeartbeat(ctx context.Context) {
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

	// 注册微服务实例
	instanceID, err := cli.RegisterInstance(svcID, config.Instance)
	if err != nil {
		log.Fatalln(err)
	}

	// 启动定时器：间隔30s
	tk := time.NewTicker(HeartbeatInterval)
	for {
		select {
		case <-tk.C:
			// 定时发送心跳
			err := cli.Heartbeat(svcID, instanceID)
			if err != nil {
				log.Println(err)
				tk.Stop()
				return
			}
			log.Println("send heartbeat success")
		// 监听程序退出
		case <-ctx.Done():
			tk.Stop()
			log.Println("service is done")
			return
		}
	}
}
