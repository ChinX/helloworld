package config

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"

	"github.com/go-yaml/yaml"
)

var (
	Service  *ServiceConf
	Instance *InstanceConf
	Registry *RegistryConf
	Provider *ServiceConf
	Tenant   *TenantConf
)

// microservice.yaml 配置
type MicroService struct {
	Service  *ServiceConf  `yaml:"service"`
	Instance *InstanceConf `yaml:"instance"`
	Registry *RegistryConf `yaml:"registry"`
	Provider *ServiceConf  `yaml:"provider"`
	Tenant   *TenantConf   `yaml:"tenant"`
}

// 微服务配置
type ServiceConf struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	AppID   string `yaml:"appId"`
}

// 实例配置
type InstanceConf struct {
	Hostname      string `yaml:"hostname"`
	Protocol      string `yaml:"protocol"`
	ListenAddress string `yaml:"listenAddress"`
}

// Service-Center 配置
type RegistryConf struct {
	Address string `yaml:"address"`
}

// 租户信息
type TenantConf struct {
	Domain string `yaml:"domain"`
}

// 加载配置
func LoadConfig(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	conf := MicroService{}

	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return err
	}

	if conf.Tenant == nil {
		conf.Tenant = &TenantConf{}
	}

	if conf.Tenant.Domain == "" {
		conf.Tenant.Domain = "default"
	}

	if conf.Instance != nil {
		if conf.Instance.Hostname == "" {
			conf.Instance.Hostname, _ = os.Hostname()
		}

		if conf.Instance.ListenAddress == "" {
			return fmt.Errorf("instance lister address is empty")
		}

		host, port, err := net.SplitHostPort(conf.Instance.ListenAddress)
		if err != nil {
			return fmt.Errorf("instance lister address is wrong: %s", err)
		}
		if host == "" {
			host = "127.0.0.1"
		}
		num, err := strconv.Atoi(port)
		if err != nil || num <= 0 {
			return fmt.Errorf("instance lister port %s is wrong: %s", port, err)
		}
		conf.Instance.ListenAddress = host + ":" + port
	}

	Service = conf.Service
	Instance = conf.Instance
	Registry = conf.Registry
	Provider = conf.Provider
	Tenant = conf.Tenant
	return nil
}
