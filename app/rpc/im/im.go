package main

import (
	"flag"
	"fmt"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/config"
	conversationserviceServer "ran-feed/app/rpc/im/internal/server/conversationservice"
	groupserviceServer "ran-feed/app/rpc/im/internal/server/groupservice"
	messageserviceServer "ran-feed/app/rpc/im/internal/server/messageservice"
	"ran-feed/app/rpc/im/internal/svc"
	"ran-feed/pkg/envx"
	"ran-feed/pkg/interceptor"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/im.yaml", "the config file")

func main() {
	flag.Parse()

	envx.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		im.RegisterConversationServiceServer(grpcServer, conversationserviceServer.NewConversationServiceServer(ctx))
		im.RegisterMessageServiceServer(grpcServer, messageserviceServer.NewMessageServiceServer(ctx))
		im.RegisterGroupServiceServer(grpcServer, groupserviceServer.NewGroupServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.ServerGrpcInterceptor())
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
