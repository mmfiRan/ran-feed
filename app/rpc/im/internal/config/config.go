package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

// Config im-rpc 配置
type Config struct {
	zrpc.RpcServerConf
	MySQL       MySQLConfig
	RedisConfig redis.RedisConf
	Message     MessageConfig
	Group       GroupConfig
}

// MySQLConfig MySQL 连接配置
type MySQLConfig struct {
	DataSource string `json:"DataSource"`
}

// MessageConfig 消息相关参数
type MessageConfig struct {
	MaxPullRange        int `json:",default=100"`
	RevokeWindowSeconds int `json:",default=120"`
}

// GroupConfig 群相关参数
type GroupConfig struct {
	MaxMemberCount int `json:",default=500"`
}
