package svc

import (
	"ran-feed/app/rpc/content/internal/common/component/contentcache"
	"ran-feed/app/rpc/content/internal/common/component/contentresolver"
	"ran-feed/app/rpc/content/internal/common/component/feedprojector"
	"ran-feed/app/rpc/content/internal/common/component/feedpub"
	"ran-feed/app/rpc/content/internal/common/component/hotfeed"
	"ran-feed/app/rpc/content/internal/common/component/publishbox"
	"ran-feed/app/rpc/content/internal/config"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/mq/producer"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/repositories/articlerepo"
	"ran-feed/app/rpc/content/internal/repositories/contentoutboxrepo"
	"ran-feed/app/rpc/content/internal/repositories/contentrepo"
	"ran-feed/app/rpc/content/internal/repositories/contentreviewrepo"
	"ran-feed/app/rpc/content/internal/repositories/videorepo"
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
	Config                  config.Config
	OssStrategy             oss.Strategy
	Redis                   *redis.Redis
	MysqlDb                 *orm.DB
	ContentRepository       repositories.ContentRepository
	ArticleRepository       repositories.ArticleRepository
	VideoRepository         repositories.VideoRepository
	ContentReviewRepository repositories.ContentReviewRepository
	ContentOutboxRepository repositories.ContentOutboxRepository
	UserRpc                 userservice.UserService
	LikesRpc                likeservice.LikeService
	FavoriteRpc             favoriteservice.FavoriteService
	FollowRpc               followservice.FollowService
	CountRpc                counterservice.CounterService
	// PublishBoxRebuildLocker 发件箱冷重建的分布式锁 大 V 发件箱是跨 pod 的热点 要防击穿
	PublishBoxRebuildLocker *cache.DistLocker
	// FollowRebuildLocker 关注流两半 inbox 与拉模式集 重建的锁 重建很贵 防前端重试和多标签页一起重建
	FollowRebuildLocker *cache.DistLocker
	// FavoriteFeedRebuildLocker 收藏流缓存重建的锁 和发布流一个量级 同样要防击穿
	FavoriteFeedRebuildLocker *cache.DistLocker
	// FeedPublisher 内容进入 feed 的发布副作用
	FeedPublisher *feedpub.Publisher
	// ContentCache 内容详情二级缓存 读回填和写失效都走它
	ContentCache *contentcache.Cache
	// FeedProjector 内容生命周期事件的 feed 投影 消息消费与对账补跑共用
	FeedProjector *feedprojector.Projector
	// HotFeed 全站热榜 算分任务与热榜读路径共用
	HotFeed *hotfeed.Feed
	// ContentResolver feed 读路径公共入口 feedservice 多个 Logic 共用
	ContentResolver *contentresolver.Resolver
	// PublishBox 作者发件箱 读命中直读 未命中回源重建
	PublishBox *publishbox.Box
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

	// 仓储先建出来 Component 和 ServiceContext 复用同一批实例
	contentRepository := contentrepo.New()
	articleRepository := articlerepo.New()
	videoRepository := videorepo.New()
	// 发件箱重建锁也是同一把 PublishBox 和 svc 复用
	publishBoxRebuildLocker := cache.NewDistLocker(redisClient)
	contentCache := contentcache.New(redisClient)
	feedPublisher := feedpub.NewPublisher(redisClient, followRpc, producer.NewFanOutProducer(kqPusher))

	return &ServiceContext{
		MysqlDb:                   mysql,
		ContentRepository:         contentRepository,
		ArticleRepository:         articleRepository,
		VideoRepository:           videoRepository,
		ContentReviewRepository:   contentreviewrepo.New(),
		ContentOutboxRepository:   contentoutboxrepo.New(),
		Config:                    c,
		OssStrategy:               ossStrategy,
		Redis:                     redisClient,
		UserRpc:                   userRpc,
		LikesRpc:                  likeRpc,
		FavoriteRpc:               favoriteRpc,
		FollowRpc:                 followRpc,
		CountRpc:                  countRpc,
		PublishBoxRebuildLocker:   publishBoxRebuildLocker,
		FollowRebuildLocker:       cache.NewDistLocker(redisClient),
		FavoriteFeedRebuildLocker: cache.NewDistLocker(redisClient),
		FeedPublisher:             feedPublisher,
		ContentCache:              contentCache,
		FeedProjector:             feedprojector.New(redisClient, contentRepository, contentCache, feedPublisher),
		HotFeed:                   hotfeed.New(redisClient, countRpc, contentRepository),
		ContentResolver: contentresolver.New(
			contentCache,
			contentRepository,
			articleRepository,
			videoRepository,
			userRpc,
			likeRpc,
		),
		PublishBox: publishbox.New(redisClient, publishBoxRebuildLocker, contentRepository),
	}
}
