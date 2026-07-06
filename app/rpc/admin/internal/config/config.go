package config

import "github.com/zeromicro/go-zero/zrpc"

type Config struct {
	zrpc.RpcServerConf
	MySQL MySQLConfig
}

type MySQLConfig struct {
	DataSource string `json:"DataSource"`
}
