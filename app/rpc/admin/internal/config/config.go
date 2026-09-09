package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf
	RedisConfig redis.RedisConf
	MySQL       MySQLConfig
	// SessionTTL 登录态过期秒数 默认 7 天
	SessionTTL int64 `json:",default=604800"`
}

type MySQLConfig struct {
	DataSource string `json:"DataSource"`
}
