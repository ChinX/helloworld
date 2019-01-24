package v3

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/apache/servicecomb-service-center/server/core/proto"
	"github.com/chinx/helloworld/rest/common/config"
	"github.com/chinx/helloworld/rest/common/restful"
)

var (
	// 接口 API 定义
	microServices = "/registry/v3/microservices"
	svcInstances  = "/registry/v3/microservices/%s/instances"
	discovery     = "/registry/v3/instances"
	existence     = "/registry/v3/existence"
	heartbeats    = "/registry/v3/heartbeats"
	watcher       = "/registry/v3/microservices/%s/watcher"

	microServiceType sourceType = "microservice"
	schemaType       sourceType = "schema"
)

type sourceType string

type Client struct {
	rawURL string
	domain string
}

func NewClient(addr string, domain string) *Client {
	return &Client{rawURL: addr, domain: domain}
}

// 查询微服务是否存在
func (c *Client) existence(params url.Values) (*proto.GetExistenceResponse, error) {
	reqURL := c.rawURL + existence + "?" + params.Encode()
	req, err := restful.NewRequest(http.MethodGet, reqURL, c.DefaultHeaders(), nil)
	if err == nil {
		respData := &proto.GetExistenceResponse{}
		err = restful.DoRequest(req, respData)
		if err == nil {
			return respData, nil
		}
	}
	return nil, err
}

// 获取微服务服务ID
func (c *Client) GetServiceID(svc *config.ServiceConf) (string, error) {
	val := url.Values{}
	val.Set("type", string(microServiceType))
	val.Set("appId", svc.AppID)
	val.Set("serviceName", svc.Name)
	val.Set("version", svc.Version)
	respData, err := c.existence(val)
	if err == nil {
		return respData.ServiceId, nil
	}
	return "", fmt.Errorf("[GetServiceID]: %s", err)
}

// 注册微服务
func (c *Client) RegisterService(svc *config.ServiceConf) (string, error) {
	ms := &proto.CreateServiceRequest{
		Service: &proto.MicroService{
			AppId:       svc.AppID,
			ServiceName: svc.Name,
			Version:     svc.Version,
		},
	}

	reqURL := c.rawURL + microServices
	req, err := restful.NewRequest(http.MethodPost, reqURL, c.DefaultHeaders(), ms)
	if err == nil {
		respData := &proto.CreateServiceResponse{}
		err = restful.DoRequest(req, respData)
		if err == nil {
			return respData.ServiceId, nil
		}
	}
	return "", fmt.Errorf("[RegisterService]: %s", err)
}

// 注册微服务实例
func (c *Client) RegisterInstance(svcID string, ins *config.InstanceConf) (string, error) {
	endpoint := ins.Protocol + "://" + ins.ListenAddress
	ms := &proto.RegisterInstanceRequest{
		Instance: &proto.MicroServiceInstance{
			HostName:  ins.Hostname,
			Endpoints: []string{endpoint},
		},
	}

	reqURL := c.rawURL + fmt.Sprintf(svcInstances, svcID)
	req, err := restful.NewRequest(http.MethodPost, reqURL, c.DefaultHeaders(), ms)
	if err == nil {
		respData := &proto.RegisterInstanceResponse{}
		err = restful.DoRequest(req, respData)
		if err == nil {
			return respData.InstanceId, nil
		}
	}
	return "", fmt.Errorf("[RegisterInstance]: %s", err)
}

// 心跳保活
func (c *Client) Heartbeat(svcID, insID string) error {
	hb := &proto.HeartbeatSetRequest{
		Instances: []*proto.HeartbeatSetElement{
			{ServiceId: svcID, InstanceId: insID},
		},
	}

	reqURL := c.rawURL + heartbeats
	req, err := restful.NewRequest(http.MethodPut, reqURL, c.DefaultHeaders(), hb)
	if err == nil {
		err = restful.DoRequest(req, nil)
	}
	if err != nil {
		return fmt.Errorf("[Heartbeat]: %s", err)
	}
	return nil
}

// 服务发现
func (c *Client) Discovery(conID string, svc *config.ServiceConf) ([]*proto.MicroServiceInstance, error) {
	val := url.Values{}
	val.Set("appId", svc.AppID)
	val.Set("serviceName", svc.Name)
	val.Set("version", svc.Version)

	reqURL := c.rawURL + discovery + "?" + val.Encode()
	req, err := restful.NewRequest(http.MethodGet, reqURL, c.DefaultHeaders(), nil)
	if err == nil {
		req.Header.Set("x-consumerid", conID)
		respData := &proto.GetInstancesResponse{}
		err = restful.DoRequest(req, respData)
		if err == nil {
			return respData.Instances, nil
		}
	}
	return nil, fmt.Errorf("[Discovery]: %s", err)
}

// 服务订阅
func (c *Client) WatchService(svcID string, callback func(*proto.WatchInstanceResponse)) error {
	addr, err:= url.Parse(c.rawURL + fmt.Sprintf(watcher, svcID))
	if err != nil {
		return fmt.Errorf("[WatchService]: parse repositry url faild: %s", err)
	}

	// 注： watch接口使用了 websocket 长连接
	addr.Scheme = "ws"
	conn, _, err := (&websocket.Dialer{}).Dial(addr.String(),c.DefaultHeaders())
	if err != nil {
		return fmt.Errorf("[WatchService]: start websocket faild: %s", err)
	}

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		if messageType == websocket.TextMessage {
			data := &proto.WatchInstanceResponse{}
			err := json.Unmarshal(message, data)
			if err != nil {
				log.Println(err)
				break
			}
			callback(data)
		}
	}
	return fmt.Errorf("[WatchService]: receive message faild: %s", err)
}

// 设置默认头部
func (c *Client) DefaultHeaders() http.Header {
	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"X-Domain-Name": []string{"default"},
	}
	if c.domain != "" {
		headers.Set("X-Domain-Name", c.domain)
	}
	return headers
}
