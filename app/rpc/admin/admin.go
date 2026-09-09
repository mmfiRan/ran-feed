package main

import (
	"flag"
	"fmt"

	"ran-feed/pkg/envx"
	"ran-feed/pkg/interceptor"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/config"
	adminauditserviceServer "ran-feed/app/rpc/admin/internal/server/adminauditservice"
	adminauthserviceServer "ran-feed/app/rpc/admin/internal/server/adminauthservice"
	adminpermissionserviceServer "ran-feed/app/rpc/admin/internal/server/adminpermissionservice"
	adminroleserviceServer "ran-feed/app/rpc/admin/internal/server/adminroleservice"
	adminuserserviceServer "ran-feed/app/rpc/admin/internal/server/adminuserservice"
	"ran-feed/app/rpc/admin/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/admin.yaml", "the config file")

func main() {
	flag.Parse()

	envx.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		admin.RegisterAdminAuthServiceServer(grpcServer, adminauthserviceServer.NewAdminAuthServiceServer(ctx))
		admin.RegisterAdminAuditServiceServer(grpcServer, adminauditserviceServer.NewAdminAuditServiceServer(ctx))
		admin.RegisterAdminPermissionServiceServer(grpcServer, adminpermissionserviceServer.NewAdminPermissionServiceServer(ctx))
		admin.RegisterAdminRoleServiceServer(grpcServer, adminroleserviceServer.NewAdminRoleServiceServer(ctx))
		admin.RegisterAdminUserServiceServer(grpcServer, adminuserserviceServer.NewAdminUserServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.ServerGrpcInterceptor())
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
