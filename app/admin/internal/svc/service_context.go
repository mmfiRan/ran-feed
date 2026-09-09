// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package svc

import (
	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/config"
	"ran-feed/app/admin/internal/middleware"
	adminauditservice "ran-feed/app/rpc/admin/client/adminauditservice"
	adminauthservice "ran-feed/app/rpc/admin/client/adminauthservice"
	adminpermissionservice "ran-feed/app/rpc/admin/client/adminpermissionservice"
	adminroleservice "ran-feed/app/rpc/admin/client/adminroleservice"
	"ran-feed/app/rpc/admin/client/adminuserservice"
	"ran-feed/app/rpc/content/client/admincontentservice"
	useradminuserservice "ran-feed/app/rpc/user/client/adminuserservice"
	"ran-feed/pkg/interceptor"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	Config                  config.Config
	Redis                   *redis.Redis
	AdminAuthRpc            adminauthservice.AdminAuthService
	AdminAuditRpc           adminauditservice.AdminAuditService
	AdminPermissionRpc      adminpermissionservice.AdminPermissionService
	AdminRoleRpc            adminroleservice.AdminRoleService
	AdminUserRpc            adminuserservice.AdminUserService
	ContentAdminRpc         admincontentservice.AdminContentService
	UserAdminRpc            useradminuserservice.AdminUserService
	AdminAuthMiddleware     rest.Middleware
	AdminRbacMiddleware     rest.Middleware
	AdminAuditMiddleware    rest.Middleware
	AdminLoginLogMiddleware rest.Middleware
}

func NewServiceContext(c config.Config) *ServiceContext {
	adminAuthRpc := adminauthservice.NewAdminAuthService(zrpc.MustNewClient(
		c.AdminRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	adminAuditRpc := adminauditservice.NewAdminAuditService(zrpc.MustNewClient(
		c.AdminRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	adminPermissionRpc := adminpermissionservice.NewAdminPermissionService(zrpc.MustNewClient(
		c.AdminRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	adminRoleRpc := adminroleservice.NewAdminRoleService(zrpc.MustNewClient(
		c.AdminRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))
	adminUserRpc := adminuserservice.NewAdminUserService(zrpc.MustNewClient(
		c.AdminRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	contentAdminRpc := admincontentservice.NewAdminContentService(zrpc.MustNewClient(
		c.ContentRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	userAdminRpc := useradminuserservice.NewAdminUserService(zrpc.MustNewClient(
		c.UserRpcClientConf,
		zrpc.WithUnaryClientInterceptor(interceptor.ClientGrpcInterceptor()),
	))

	r := redis.MustNewRedis(c.RedisConfig)

	return &ServiceContext{
		Config:                  c,
		Redis:                   r,
		AdminAuthRpc:            adminAuthRpc,
		AdminAuditRpc:           adminAuditRpc,
		AdminPermissionRpc:      adminPermissionRpc,
		AdminRoleRpc:            adminRoleRpc,
		AdminUserRpc:            adminUserRpc,
		ContentAdminRpc:         contentAdminRpc,
		UserAdminRpc:            userAdminRpc,
		AdminAuthMiddleware:     middleware.NewAdminAuthMiddleware(r, c).Handle,
		AdminRbacMiddleware:     middleware.NewAdminRbacMiddleware(r, adminAuthRpc, consts.RedisAdminPermExpireSeconds).Handle,
		AdminAuditMiddleware:    middleware.NewAdminAuditMiddleware(adminAuditRpc).Handle,
		AdminLoginLogMiddleware: middleware.NewAdminLoginLogMiddleware(adminAuditRpc).Handle,
	}
}
