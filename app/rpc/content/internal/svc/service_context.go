package svc

import (
	"ran-feed/app/rpc/content/internal/common/component/bigv"
	"ran-feed/app/rpc/content/internal/common/component/contentcache"
	"ran-feed/app/rpc/content/internal/common/component/contentresolver"
	"ran-feed/app/rpc/content/internal/common/component/favoritebox"
	"ran-feed/app/rpc/content/internal/common/component/feedprojector"
	"ran-feed/app/rpc/content/internal/common/component/feedpub"
	"ran-feed/app/rpc/content/internal/common/component/followfeed"
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
	// BigV 全局大 V 集合 判推拉分流
	BigV *bigv.Set
	// ContentCache 内容详情二级缓存 读回填与写失效的 owner
	ContentCache *contentcache.Cache
	// PublishBox 作者发件箱 owner
	PublishBox *publishbox.Box
	// FollowFeed 关注流两半 收件箱与拉模式集的 owner
	FollowFeed *followfeed.Feed
	// FavoriteBox 用户收藏流 owner
	FavoriteBox *favoritebox.Box
	// HotFeed 全站热榜 owner 算分任务与读路径共用
	HotFeed *hotfeed.Feed
	// FeedPublisher 内容进入 feed 的发布副作用编排
	FeedPublisher *feedpub.Publisher
	// FeedProjector 内容生命周期事件的 feed 投影 消息消费与对账补跑共用
	FeedProjector *feedprojector.Projector
	// ContentResolver feed 读路径公共入口 feedservice 多个 Logic 共用
	ContentResolver *contentresolver.Resolver
}

func NewServiceContext(c config.Config) *ServiceContext {
	// 基础设施客户端 query.SetDefault 必须在任何 Repository 被使用之前只执行一次
	mysql := orm.MustNewMysql(&orm.Config{DSN: c.MySQL.DataSource})
	query.SetDefault(mysql.DB)
	redisClient := redis.MustNewRedis(c.RedisConfig)

	// OSS 直传凭证策略
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

	// RPC 客户端
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

	// 仓储
	contentRepository := contentrepo.New()
	articleRepository := articlerepo.New()
	videoRepository := videorepo.New()

	// 扇出生产者 Hash balancer 配合 PushWithKey 保证同内容分批落同分区
	fanOutProducer := producer.NewFanOutProducer(kq.NewPusher(
		c.KqFanOutProducerConf.Brokers,
		c.KqFanOutProducerConf.Topic,
		kq.WithBalancer(&kafka.Hash{}),
	))

	// 组件 每份状态一个 owner 重建锁跟着它保护的那份状态走
	bigvSet := bigv.New(redisClient)
	contentCache := contentcache.New(redisClient)
	publishBox := publishbox.New(redisClient, cache.NewDistLocker(redisClient), contentRepository)
	followFeed := followfeed.New(redisClient, cache.NewDistLocker(redisClient), followRpc, bigvSet, contentRepository)
	favoriteBox := favoritebox.New(redisClient, cache.NewDistLocker(redisClient), favoriteRpc)
	hotFeed := hotfeed.New(redisClient, countRpc, contentRepository)
	feedPublisher := feedpub.NewPublisher(redisClient, followRpc, bigvSet, publishBox, fanOutProducer)

	return &ServiceContext{
		Config:                  c,
		OssStrategy:             ossStrategy,
		Redis:                   redisClient,
		MysqlDb:                 mysql,
		ContentRepository:       contentRepository,
		ArticleRepository:       articleRepository,
		VideoRepository:         videoRepository,
		ContentReviewRepository: contentreviewrepo.New(),
		ContentOutboxRepository: contentoutboxrepo.New(),
		UserRpc:                 userRpc,
		LikesRpc:                likeRpc,
		FavoriteRpc:             favoriteRpc,
		FollowRpc:               followRpc,
		CountRpc:                countRpc,
		BigV:                    bigvSet,
		ContentCache:            contentCache,
		PublishBox:              publishBox,
		FollowFeed:              followFeed,
		FavoriteBox:             favoriteBox,
		HotFeed:                 hotFeed,
		FeedPublisher:           feedPublisher,
		FeedProjector:           feedprojector.New(contentRepository, contentCache, hotFeed, publishBox, feedPublisher),
		ContentResolver: contentresolver.New(
			contentCache,
			contentRepository,
			articleRepository,
			videoRepository,
			userRpc,
			likeRpc,
		),
	}
}
