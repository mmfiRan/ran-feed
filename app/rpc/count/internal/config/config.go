package config

import (
	"time"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	RedisConfig    redis.RedisConf
	MySQL          MySQLConfig
	KqConsumerConf kq.KqConf
	XxlJob         XxlJobConfig
}

type (
	MySQLConfig struct {
		DataSource string `json:"DataSource"`
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
