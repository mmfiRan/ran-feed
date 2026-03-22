// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"ran-feed/app/front/internal/middleware"
	"strings"

	"ran-feed/app/front/internal/common/consts"
	"ran-feed/app/front/internal/common/oss"
	"ran-feed/app/front/internal/common/oss/strategy"
	"ran-feed/app/front/internal/config"
	"ran-feed/app/rpc/content/client/contentservice"
	"ran-feed/app/rpc/content/client/feedservice"
	"ran-feed/app/rpc/count/client/counterservice"
	"ran-feed/app/rpc/interaction/client/commentservice"
	"ran-feed/app/rpc/interaction/client/favoriteservice"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/app/rpc/interaction/client/likeservice"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/interceptor"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config                        config.Config
	Redis                         *redis.Redis
	UserLoginStatusAuthMiddleware rest.Middleware
	OptionalLoginMiddleware       rest.Middleware
	ContentRpc                    contentservice.ContentService
	FeedRpc                       feedservice.FeedService
	LikeRpc                       likeservice.LikeService
	FavoriteRpc                   favoriteservice.FavoriteService
	CommentRpc                    commentservice.CommentService
	FollowRpc                     followservice.FollowService
	UserRpc                       userservice.UserService
	CountRpc                      counterservice.CounterService
	OssContext                    *oss.Context
}

func NewServiceContext(c config.Config) *ServiceContext {
	contentRpc := contentservice.NewContentService(zrpc.MustNewClient(
		c.ContentRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	feedRpc := feedservice.NewFeedService(zrpc.MustNewClient(
		c.ContentRpcClientConf,
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
	commentRpc := commentservice.NewCommentService(zrpc.MustNewClient(
		c.InteractionRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	followRpc := followservice.NewFollowService(zrpc.MustNewClient(
		c.InteractionRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	userRpc := userservice.NewUserService(zrpc.MustNewClient(
		c.UserRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	countRpc := counterservice.NewCounterService(zrpc.MustNewClient(
		c.CountRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

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

	r := redis.MustNewRedis(c.RedisConfig)

	return &ServiceContext{
		Config:                        c,
		ContentRpc:                    contentRpc,
		FeedRpc:                       feedRpc,
		LikeRpc:                       likeRpc,
		FavoriteRpc:                   favoriteRpc,
		CommentRpc:                    commentRpc,
		FollowRpc:                     followRpc,
		UserRpc:                       userRpc,
		CountRpc:                      countRpc,
		OssContext:                    ossContext,
		Redis:                         r,
		UserLoginStatusAuthMiddleware: middleware.NewUserLoginStatusAuthMiddleware(r, c).Handle,
		OptionalLoginMiddleware:       middleware.NewOptionalLoginMiddleware(r, c).Handle,
	}
}
