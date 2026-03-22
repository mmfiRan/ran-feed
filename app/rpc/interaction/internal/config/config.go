package config

import (
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	RedisConfig          redis.RedisConf
	KqProducerConf       KqProducerConf
	KqConsumerConf       kq.KqConf
	MySQL                MySQLConfig
	CountRpcClientConf   zrpc.RpcClientConf
	UserRpcClientConf    zrpc.RpcClientConf
	ContentRpcClientConf zrpc.RpcClientConf
}

type (
	KqProducerConf struct {
		Brokers    []string `json:"Brokers"`
		Topic      string   `json:"Topic"`
		MaxRetries int      `json:"MaxRetries,default=3"` // 最大重试次数
	}
	MySQLConfig struct {
		DataSource string `json:"DataSource"`
	}
)
