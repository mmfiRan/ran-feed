package svc

import (
	"github.com/elastic/go-elasticsearch/v8"

	"ran-feed/app/rpc/search/internal/config"
	"ran-feed/app/rpc/search/internal/entity/query"
	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/pkg/orm"
)

type ServiceContext struct {
	Config  config.Config
	ES      *elasticsearch.Client
	MysqlDb *orm.DB
}

func NewServiceContext(c config.Config) *ServiceContext {
	mysql := orm.MustNewMysql(&orm.Config{DSN: c.MySQL.DataSource})
	query.SetDefault(mysql.DB)
	return &ServiceContext{
		Config:  c,
		ES:      es.MustNewClient(c.Elasticsearch.Addresses, c.Elasticsearch.Username, c.Elasticsearch.Password),
		MysqlDb: mysql,
	}
}
