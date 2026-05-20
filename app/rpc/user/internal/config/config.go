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
	LoginRateLimit           LoginRateLimitConfig
	UserCache                UserCacheConfig
	InteractionRpcClientConf zrpc.RpcClientConf
	CountRpcClientConf       zrpc.RpcClientConf
}

// LoginRateLimitConfig 登录频率限制（按手机号维度，滑动窗口）。
// WindowSeconds 或 MaxAttempts <= 0 时整体关闭。
type LoginRateLimitConfig struct {
	WindowSeconds int64 `json:",default=300"`
	MaxAttempts   int64 `json:",default=5"`
}

// UserCacheConfig 用户信息缓存配置。TTLSeconds<=0 时整体关闭缓存。
// 正负 TTL 均叠加 [0, JitterMaxSeconds] 抖动以抗雪崩。
type UserCacheConfig struct {
	TTLSeconds               int64 `json:",default=600"`
	NegativeTTLSeconds       int64 `json:",default=60"`
	JitterMaxSeconds         int64 `json:",default=600"`
	NegativeJitterMaxSeconds int64 `json:",default=60"`
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
