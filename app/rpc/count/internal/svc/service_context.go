package svc

import (
	"ran-feed/app/rpc/count/internal/config"
	"ran-feed/app/rpc/count/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config  config.Config
	Redis   *redis.Redis
	MysqlDb *orm.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	ormConfig := &orm.Config{
		DSN: c.MySQL.DataSource,
	}
	mysql := orm.MustNewMysql(ormConfig)
	query.SetDefault(mysql.DB)
	return &ServiceContext{
		Config:  c,
		Redis:   redis.MustNewRedis(c.RedisConfig),
		MysqlDb: mysql,
	}
}
