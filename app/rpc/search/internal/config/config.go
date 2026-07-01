package config

import (
	"time"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	MySQL                MySQLConfig
	Elasticsearch        ESConfig
	KqConsumerConf       kq.KqConf
	XxlJob               XxlJobConfig
	ContentRpcClientConf zrpc.RpcClientConf
	UserRpcClientConf    zrpc.RpcClientConf
}

type (
	MySQLConfig struct {
		DataSource string
	}

	ESConfig struct {
		Addresses []string
		Username  string
		Password  string
	}

	XxlJobConfig struct {
		AppName          string
		Address          string
		IP               string
		Port             int
		AccessToken      string
		AdminAddresses   []string
		RegistryInterval time.Duration
		HTTPTimeout      time.Duration
	}
)
