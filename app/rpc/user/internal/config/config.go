package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	RedisConfig              redis.RedisConf
	Oss                      OssConfig
	MySQL                    MySQLConfig
	SessionTTL               int64
	InteractionRpcClientConf zrpc.RpcClientConf
	CountRpcClientConf       zrpc.RpcClientConf
}

type OssConfig struct {
	Provider        string `json:",env=OSS_PROVIDER"`
	Region          string `json:",env=OSS_REGION"`
	BucketName      string `json:",env=OSS_BUCKET_NAME"`
	AccessKeyId     string `json:",env=OSS_ACCESS_KEY_ID"`
	AccessKeySecret string `json:",env=OSS_ACCESS_KEY_SECRET"`
	Endpoint        string `json:",env=OSS_ENDPOINT"`
	UploadDir       string `json:",env=OSS_UPLOAD_DIR"`
	PublicHost      string `json:",env=OSS_PUBLIC_HOST"`
}

type MySQLConfig struct {
	DataSource string `json:"DataSource"`
}
