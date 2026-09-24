package svc

import (
	"ran-feed/app/rpc/content/internal/config"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/feedpub"
	"ran-feed/app/rpc/content/internal/mq/producer"
	"ran-feed/app/rpc/count/client/counterservice"
	"ran-feed/app/rpc/interaction/client/favoriteservice"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/app/rpc/interaction/client/likeservice"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/cache"
	"ran-feed/pkg/interceptor"
	"ran-feed/pkg/orm"
	"ran-feed/pkg/oss"
	"ran-feed/pkg/oss/aliyun"

	"github.com/segmentio/kafka-go"
	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/zrpc"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config      config.Config
	OssStrategy oss.Strategy
	Redis       *redis.Redis
	MysqlDb     *orm.DB
	UserRpc     userservice.UserService
	LikesRpc    likeservice.LikeService
	FavoriteRpc favoriteservice.FavoriteService
	FollowRpc   followservice.FollowService
	CountRpc    counterservice.CounterService
	// PublishBoxRebuildLocker 发件箱冷重建分布式锁 大V发件箱属跨 pod 热点 防击穿
	PublishBoxRebuildLocker *cache.DistLocker
	// FollowRebuildLocker 关注流两半(inbox 与拉模式集)重建分布式锁 重建成本极高 防前端重试与多标签页并发重建
	FollowRebuildLocker *cache.DistLocker
	// FavoriteFeedRebuildLocker 收藏流缓存重建分布式锁 与发布流同级防击穿
	FavoriteFeedRebuildLocker *cache.DistLocker
	// FeedPublisher 内容发布
	FeedPublisher *feedpub.Publisher
}

func NewServiceContext(c config.Config) *ServiceContext {

	// 初始化MySQL
	ormConfig := &orm.Config{
		DSN: c.MySQL.DataSource,
	}
	mysql := orm.MustNewMysql(ormConfig)
	query.SetDefault(mysql.DB)

	// 初始化OSS 直传凭证策略
	ossStrategy := aliyun.New(aliyun.Config{
		Region:          c.Oss.Region,
		BucketName:      c.Oss.BucketName,
		AccessKeyID:     c.Oss.AccessKeyId,
		AccessKeySecret: c.Oss.AccessKeySecret,
		RoleArn:         c.Oss.RoleArn,
		RoleSessionName: c.Oss.RoleSessionName,
		DurationSeconds: c.Oss.DurationSeconds,
		UploadDir:       c.Oss.UploadDir,
	})

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
	redisClient := redis.MustNewRedis(c.RedisConfig)
	// Kafka 扇出生产者 Hash balancer 配合 PushWithKey 保证同内容分批落同分区
	kqPusher := kq.NewPusher(
		c.KqFanOutProducerConf.Brokers,
		c.KqFanOutProducerConf.Topic,
		kq.WithBalancer(&kafka.Hash{}),
	)
	return &ServiceContext{
		MysqlDb:                   mysql,
		Config:                    c,
		OssStrategy:               ossStrategy,
		Redis:                     redisClient,
		UserRpc:                   userRpc,
		LikesRpc:                  likeRpc,
		FavoriteRpc:               favoriteRpc,
		FollowRpc:                 followRpc,
		CountRpc:                  countRpc,
		PublishBoxRebuildLocker:   cache.NewDistLocker(redisClient),
		FollowRebuildLocker:       cache.NewDistLocker(redisClient),
		FavoriteFeedRebuildLocker: cache.NewDistLocker(redisClient),
		FeedPublisher:             feedpub.NewPublisher(redisClient, followRpc, producer.NewFanOutProducer(kqPusher)),
	}
}
