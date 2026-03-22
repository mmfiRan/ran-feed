package svc

import (
	"ran-feed/app/rpc/content/internal/common/consts"
	"ran-feed/app/rpc/content/internal/common/oss"
	"ran-feed/app/rpc/content/internal/common/oss/strategy"
	"ran-feed/app/rpc/content/internal/config"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/count/client/counterservice"
	"ran-feed/app/rpc/interaction/client/favoriteservice"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/app/rpc/interaction/client/likeservice"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/interceptor"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/zrpc"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config      config.Config
	OssContext  *oss.Context
	Redis       *redis.Redis
	MysqlDb     *orm.DB
	UserRpc     userservice.UserService
	LikesRpc    likeservice.LikeService
	FavoriteRpc favoriteservice.FavoriteService
	FollowRpc   followservice.FollowService
	CountRpc    counterservice.CounterService
}

func NewServiceContext(c config.Config) *ServiceContext {

	// 初始化MySQL
	ormConfig := &orm.Config{
		DSN: c.MySQL.DataSource,
	}
	mysql := orm.MustNewMysql(ormConfig)
	query.SetDefault(mysql.DB)

	// 初始化OSS
	factory := oss.NewStrategyFactory()
	factory.Register(consts.Aliyun, strategy.NewAliyunStrategy(&strategy.AliyunConfig{
		Region:          c.Oss.Region,
		BucketName:      c.Oss.BucketName,
		AccessKeyId:     c.Oss.AccessKeyId,
		AccessKeySecret: c.Oss.AccessKeySecret,
		RoleArn:         c.Oss.RoleArn,
		RoleSessionName: c.Oss.RoleSessionName,
		DurationSeconds: c.Oss.DurationSeconds,
		UploadDir:       c.Oss.UploadDir,
	}))
	ossStrategy := factory.MustGetStrategy(c.Oss.Provider)
	ossContext := oss.NewContext(ossStrategy)

	userRpc := userservice.NewUserService(zrpc.MustNewClient(
		c.UserRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	likeRpc := likeservice.NewLikeService(zrpc.MustNewClient(
		c.InteractionRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	favoriteRpc := favoriteservice.NewFavoriteService(zrpc.MustNewClient(
		c.InteractionRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	followRpc := followservice.NewFollowService(zrpc.MustNewClient(
		c.InteractionRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	countRpc := counterservice.NewCounterService(zrpc.MustNewClient(
		c.CountRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	return &ServiceContext{
		MysqlDb:     mysql,
		Config:      c,
		OssContext:  ossContext,
		Redis:       redis.MustNewRedis(c.RedisConfig),
		UserRpc:     userRpc,
		LikesRpc:    likeRpc,
		FavoriteRpc: favoriteRpc,
		FollowRpc:   followRpc,
		CountRpc:    countRpc,
	}
}
