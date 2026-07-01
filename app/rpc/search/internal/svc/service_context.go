package svc

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/zeromicro/go-zero/zrpc"

	"ran-feed/app/rpc/content/client/feedservice"
	"ran-feed/app/rpc/search/internal/config"
	"ran-feed/app/rpc/search/internal/entity/query"
	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/interceptor"
	"ran-feed/pkg/orm"
)

type ServiceContext struct {
	Config     config.Config
	ES         *elasticsearch.Client
	MysqlDb    *orm.DB
	ContentRpc feedservice.FeedService
	UserRpc    userservice.UserService
}

func NewServiceContext(c config.Config) *ServiceContext {
	mysql := orm.MustNewMysql(&orm.Config{DSN: c.MySQL.DataSource})
	query.SetDefault(mysql.DB)
	return &ServiceContext{
		Config:  c,
		ES:      es.MustNewClient(c.Elasticsearch.Addresses, c.Elasticsearch.Username, c.Elasticsearch.Password),
		MysqlDb: mysql,
		ContentRpc: feedservice.NewFeedService(zrpc.MustNewClient(
			c.ContentRpcClientConf,
			zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
		)),
		UserRpc: userservice.NewUserService(zrpc.MustNewClient(
			c.UserRpcClientConf,
			zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
		)),
	}
}
