package svc

import (
	"strings"

	"ran-feed/app/rpc/count/client/counterservice"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/app/rpc/user/internal/common/consts"
	"ran-feed/app/rpc/user/internal/common/oss"
	"ran-feed/app/rpc/user/internal/common/oss/strategy"
	"ran-feed/app/rpc/user/internal/config"
	"ran-feed/app/rpc/user/internal/entity/query"
	"ran-feed/pkg/interceptor"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config     config.Config
	OssContext *oss.Context
	Redis      *redis.Redis
	MysqlDb    *orm.DB
	FollowRpc  followservice.FollowService
	CountRpc   counterservice.CounterService
}

func NewServiceContext(c config.Config) *ServiceContext {
	ormConfig := &orm.Config{
		DSN: c.MySQL.DataSource,
	}
	mysql := orm.MustNewMysql(ormConfig)
	query.SetDefault(mysql.DB)

	provider := strings.TrimSpace(c.Oss.Provider)
	if provider == "" {
		provider = consts.Aliyun
	}
	factory := oss.NewStrategyFactory()
	factory.Register(consts.Aliyun, strategy.NewAliyunStrategy(&strategy.AliyunConfig{
		Region:          c.Oss.Region,
		BucketName:      c.Oss.BucketName,
		AccessKeyId:     c.Oss.AccessKeyId,
		AccessKeySecret: c.Oss.AccessKeySecret,
		Endpoint:        c.Oss.Endpoint,
		UploadDir:       c.Oss.UploadDir,
		PublicHost:      c.Oss.PublicHost,
	}))
	ossStrategy := factory.MustGetStrategy(provider)
	ossContext := oss.NewContext(ossStrategy)
	followRpc := followservice.NewFollowService(zrpc.MustNewClient(
		c.InteractionRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	countRpc := counterservice.NewCounterService(zrpc.MustNewClient(
		c.CountRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	return &ServiceContext{
		Config:     c,
		OssContext: ossContext,
		Redis:      redis.MustNewRedis(c.RedisConfig),
		MysqlDb:    mysql,
		FollowRpc:  followRpc,
		CountRpc:   countRpc,
	}
}
