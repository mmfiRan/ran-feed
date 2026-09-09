package svc

import (
	"ran-feed/app/rpc/content/client/contentservice"
	"ran-feed/app/rpc/count/client/counterservice"
	"ran-feed/app/rpc/interaction/internal/config"
	"ran-feed/app/rpc/interaction/internal/entity/query"
	"ran-feed/app/rpc/interaction/internal/mq/producer"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/cache"
	"ran-feed/pkg/interceptor"
	"ran-feed/pkg/orm"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config       config.Config
	Redis        *redis.Redis
	KqProducer   *kq.Pusher
	LikeProducer *producer.LikeProducer
	MysqlDb      *orm.DB
	CountRpc     counterservice.CounterService
	UserRpc      userservice.UserService
	ContentRpc   contentservice.ContentService
	// LikeUserRebuildLocker 用户点赞 hash 缓存重建分布式锁 防跨实例并发重复打 DB
	LikeUserRebuildLocker *cache.DistLocker
}

func NewServiceContext(c config.Config) *ServiceContext {
	countRpc := counterservice.NewCounterService(zrpc.MustNewClient(
		c.CountRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	userRpc := userservice.NewUserService(zrpc.MustNewClient(
		c.UserRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	contentRpc := contentservice.NewContentService(zrpc.MustNewClient(
		c.ContentRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	// 初始化MySQL
	ormConfig := &orm.Config{
		DSN: c.MySQL.DataSource,
	}
	mysql := orm.MustNewMysql(ormConfig)
	query.SetDefault(mysql.DB)

	// 初始化 Kafka 生产者 Hash balancer 配合 PushWithKey 保证同 key 路由到同分区
	kqPusher := kq.NewPusher(
		c.KqProducerConf.Brokers,
		c.KqProducerConf.Topic,
		kq.WithBalancer(&kafka.Hash{}),
	)

	maxRetries := c.KqProducerConf.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	redisClient := redis.MustNewRedis(c.RedisConfig)
	return &ServiceContext{
		Config:                c,
		Redis:                 redisClient,
		KqProducer:            kqPusher,
		LikeProducer:          producer.NewLikeProducer(kqPusher, maxRetries),
		MysqlDb:               mysql,
		CountRpc:              countRpc,
		UserRpc:               userRpc,
		ContentRpc:            contentRpc,
		LikeUserRebuildLocker: cache.NewDistLocker(redisClient),
	}
}
