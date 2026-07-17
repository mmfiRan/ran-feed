package main

import (
	"context"
	"flag"
	"fmt"

	"ran-feed/app/rpc/notification/internal/config"
	"ran-feed/app/rpc/notification/internal/mq/consumer"
	notificationserviceServer "ran-feed/app/rpc/notification/internal/server/notificationservice"
	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"
	"ran-feed/pkg/envx"
	"ran-feed/pkg/interceptor"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/notification.yaml", "the config file")

func main() {
	flag.Parse()

	envx.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		notification.RegisterNotificationServiceServer(grpcServer, notificationserviceServer.NewNotificationServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.ServerGrpcInterceptor())

	// 起 gRPC + Canal 消费者服务组
	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	for _, mq := range consumer.Consumers(c, context.Background(), ctx) {
		serviceGroup.Add(mq)
	}
	serviceGroup.Add(s)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	if c.KqConsumerConf.Topic != "" {
		fmt.Printf("Starting canal mq consumer for topic: %s group: %s...\n", c.KqConsumerConf.Topic, c.KqConsumerConf.Group)
	}
	serviceGroup.Start()
}
