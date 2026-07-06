package svc

import (
	"ran-feed/app/rpc/admin/internal/config"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"
)

type ServiceContext struct {
	Config  config.Config
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
		MysqlDb: mysql,
	}
}
