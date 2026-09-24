package config

import (
	"time"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	RedisConfig              redis.RedisConf
	Oss                      OssConfig
	MySQL                    MySQLConfig
	XxlJob                   XxlJobConfig
	UserRpcClientConf        zrpc.RpcClientConf
	InteractionRpcClientConf zrpc.RpcClientConf
	CountRpcClientConf       zrpc.RpcClientConf
	KqConsumerConf           kq.KqConf
	KqFanOutProducerConf     KqProducerConf
	KqFanOutConsumerConf     kq.KqConf
}

// KqProducerConf Kafka 生产者配置
type KqProducerConf struct {
	Brokers    []string
	Topic      string
	MaxRetries int `json:",default=3"`
}

type OssConfig struct {
	Region          string
	BucketName      string
	AccessKeyId     string
	AccessKeySecret string
	RoleArn         string
	RoleSessionName string
	DurationSeconds int64
	UploadDir       string
}
type MySQLConfig struct {
	DataSource string
}

type XxlJobConfig struct {
	AppName          string
	Address          string
	IP               string
	Port             int
	AccessToken      string
	AdminAddresses   []string
	RegistryInterval time.Duration
	HTTPTimeout      time.Duration
}
