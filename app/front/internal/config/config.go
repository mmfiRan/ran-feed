// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	ContentRpcClientConf     zrpc.RpcClientConf
	InteractionRpcClientConf zrpc.RpcClientConf
	UserRpcClientConf        zrpc.RpcClientConf
	CountRpcClientConf       zrpc.RpcClientConf
	Oss                      OssConfig
	SessionTTL               int64
	RedisConfig              redis.RedisConf
}

type OssConfig struct {
	Provider        string
	Region          string
	BucketName      string
	AccessKeyId     string
	AccessKeySecret string
	Endpoint        string
	UploadDir       string
	PublicHost      string
}
