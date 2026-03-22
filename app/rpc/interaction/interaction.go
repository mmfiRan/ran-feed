package main

import (
	"context"
	"flag"
	"fmt"
	"ran-feed/app/rpc/interaction/internal/mq/consumer"
	commentServer "ran-feed/app/rpc/interaction/internal/server/commentservice"
	"ran-feed/pkg/envx"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/config"
	favoriteServer "ran-feed/app/rpc/interaction/internal/server/favoriteservice"
	followServer "ran-feed/app/rpc/interaction/internal/server/followservice"
	likeServer "ran-feed/app/rpc/interaction/internal/server/likeservice"

	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/interceptor"
)

var configFile = flag.String("f", "etc/interaction.yaml", "the config file")

func main() {
	flag.Parse()

	envx.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	// 启动 gRPC 服务
	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		interaction.RegisterFavoriteServiceServer(grpcServer, favoriteServer.NewFavoriteServiceServer(ctx))
		interaction.RegisterFollowServiceServer(grpcServer, followServer.NewFollowServiceServer(ctx))
		interaction.RegisterLikeServiceServer(grpcServer, likeServer.NewLikeServiceServer(ctx))
		interaction.RegisterCommentServiceServer(grpcServer, commentServer.NewCommentServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.ServerGrpcInterceptor())
	// 启动消费者服务组
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()

	// 注册消费者
	mqs := consumer.Consumers(c, context.Background(), ctx)
	for _, mq := range mqs {
		serviceGroup.Add(mq)
	}
	serviceGroup.Add(s)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	fmt.Printf("Starting mq consumer for topic: %s, group: %s...\n", c.KqConsumerConf.Topic, c.KqConsumerConf.Group)
	serviceGroup.Start()
}
