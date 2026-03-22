package main

import (
	"context"
	"flag"
	"fmt"
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/config"
	"ran-feed/app/rpc/count/internal/mq/consumer"
	counterserviceServer "ran-feed/app/rpc/count/internal/server/counterservice"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/envx"
	"ran-feed/pkg/interceptor"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/count.yaml", "the config file")

func main() {
	flag.Parse()

	envx.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		count.RegisterCounterServiceServer(grpcServer, counterserviceServer.NewCounterServiceServer(ctx))

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
	if c.KqConsumerConf.Topic != "" {
		fmt.Printf("Starting canal mq consumer for topic: %s, group: %s...\n", c.KqConsumerConf.Topic, c.KqConsumerConf.Group)
	}
	serviceGroup.Start()
}
