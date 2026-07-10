// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/config"
	"ran-feed/app/admin/internal/middleware"
	"ran-feed/app/rpc/admin/client/adminservice"
	"ran-feed/app/rpc/content/client/admincontentservice"
	"ran-feed/app/rpc/user/client/adminuserservice"
	"ran-feed/pkg/interceptor"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config               config.Config
	Redis                *redis.Redis
	AdminRpc             adminservice.AdminService
	ContentAdminRpc      admincontentservice.AdminContentService
	UserAdminRpc         adminuserservice.AdminUserService
	AdminAuthMiddleware  rest.Middleware
	AdminRbacMiddleware  rest.Middleware
	AdminAuditMiddleware rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	adminRpc := adminservice.NewAdminService(zrpc.MustNewClient(
		c.AdminRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	contentAdminRpc := admincontentservice.NewAdminContentService(zrpc.MustNewClient(
		c.ContentRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	userAdminRpc := adminuserservice.NewAdminUserService(zrpc.MustNewClient(
		c.UserRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	r := redis.MustNewRedis(c.RedisConfig)

	return &ServiceContext{
		Config:               c,
		Redis:                r,
		AdminRpc:             adminRpc,
		ContentAdminRpc:      contentAdminRpc,
		UserAdminRpc:         userAdminRpc,
		AdminAuthMiddleware:  middleware.NewAdminAuthMiddleware(r, c).Handle,
		AdminRbacMiddleware:  middleware.NewAdminRbacMiddleware(r, adminRpc, consts.RedisAdminPermExpireSeconds).Handle,
		AdminAuditMiddleware: middleware.NewAdminAuditMiddleware(adminRpc).Handle,
	}
}
