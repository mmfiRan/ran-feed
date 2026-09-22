package svc

import (
	"ran-feed/app/rpc/count/client/counterservice"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/app/rpc/user/internal/config"
	"ran-feed/app/rpc/user/internal/entity/query"
	"ran-feed/pkg/interceptor"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config    config.Config
	Redis     *redis.Redis
	MysqlDb   *orm.DB
	FollowRpc followservice.FollowService
	CountRpc  counterservice.CounterService
}

func NewServiceContext(c config.Config) *ServiceContext {
	ormConfig := &orm.Config{
		DSN: c.MySQL.DataSource,
	}
	mysql := orm.MustNewMysql(ormConfig)
	query.SetDefault(mysql.DB)

	followRpc := followservice.NewFollowService(zrpc.MustNewClient(
		c.InteractionRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	countRpc := counterservice.NewCounterService(zrpc.MustNewClient(
		c.CountRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	return &ServiceContext{
		Config:    c,
		Redis:     redis.MustNewRedis(c.RedisConfig),
		MysqlDb:   mysql,
		FollowRpc: followRpc,
		CountRpc:  countRpc,
	}
}
