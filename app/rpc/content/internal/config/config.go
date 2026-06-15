package config

import (
	"time"

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
	FollowFanOut             FollowFanOutConfig
	ContentCache             ContentCacheConfig
}

// ContentCacheConfig feed 二级缓存内容详情配置 TTLSeconds<=0 时整体关闭缓存
// 正负 TTL 均叠加 [0 JitterMaxSeconds] 抖动抗雪崩
type ContentCacheConfig struct {
	TTLSeconds               int64 `json:",default=600"`
	NegativeTTLSeconds       int64 `json:",default=60"`
	JitterMaxSeconds         int64 `json:",default=600"`
	NegativeJitterMaxSeconds int64 `json:",default=60"`
}

// FollowFanOutConfig 关注流推送配置（推拉结合）
// 字段为 0 时由代码使用内置默认值
type FollowFanOutConfig struct {
	BigVFollowerThreshold  int64 // 大 V 阈值：粉丝数 >= 此值的作者发布时不推送（默认 5000）
	BatchSize              int   // 单次 ListFollowers 拉取批大小（默认 500）
	InboxKeepN             int64 // follower inbox 保留最近条数（默认 5000）
	BigVMergeMaxQuery      int   // 读路径单次请求查询大 V 数硬上限（默认 100）
	BigVCacheTTLSeconds    int   // viewer 大 V 列表缓存 TTL 秒数（默认 300）
	BigVFolloweesScanLimit int   // 计算 viewer 大 V 列表时扫描关注上限（默认 500）
	BigVCountConcurrency   int   // 计算大 V 时并行 GetCount 度（默认 16）
}

type OssConfig struct {
	Provider        string
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
