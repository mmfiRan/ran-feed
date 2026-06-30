package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	MySQL         MySQLConfig
	Elasticsearch ESConfig
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
)
