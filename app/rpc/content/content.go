package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/config"
	"ran-feed/app/rpc/content/internal/cron"
	"ran-feed/app/rpc/content/internal/mq"
	admincontentserviceServer "ran-feed/app/rpc/content/internal/server/admincontentservice"
	contentserviceServer "ran-feed/app/rpc/content/internal/server/contentservice"
	feedserviceServer "ran-feed/app/rpc/content/internal/server/feedservice"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/envx"
	"ran-feed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"ran-feed/pkg/interceptor"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/content.yaml", "the config file")

func main() {
	flag.Parse()
	envx.Load()
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		content.RegisterContentServiceServer(grpcServer, contentserviceServer.NewContentServiceServer(ctx))
		content.RegisterFeedServiceServer(grpcServer, feedserviceServer.NewFeedServiceServer(ctx))
		content.RegisterAdminContentServiceServer(grpcServer, admincontentserviceServer.NewAdminContentServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.ServerGrpcInterceptor())

	xxlCtx, cancelXxl := context.WithCancel(context.Background())
	defer cancelXxl()

	executor := xxljob.NewExecutor(xxljob.Config{
		AppName:          c.XxlJob.AppName,
		Address:          c.XxlJob.Address,
		IP:               c.XxlJob.IP,
		Port:             c.XxlJob.Port,
		AccessToken:      c.XxlJob.AccessToken,
		AdminAddresses:   c.XxlJob.AdminAddresses,
		RegistryInterval: c.XxlJob.RegistryInterval,
		HTTPTimeout:      c.XxlJob.HTTPTimeout,
	})
	cron.Register(xxlCtx, executor, ctx)
	threading.GoSafe(func() {
		if err := executor.Start(xxlCtx); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			logx.Errorf("xxl-job server 启动失败,err:%v", err)
		}
	})

	serviceGroup := service.NewServiceGroup()
	defer serviceGroup.Stop()
	for _, mq := range mq.Consumers(c, context.Background(), ctx) {
		serviceGroup.Add(mq)
	}
	serviceGroup.Add(s)

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	if c.KqConsumerConf.Topic != "" {
		fmt.Printf("Starting content outbox consumer for topic: %s group: %s...\n", c.KqConsumerConf.Topic, c.KqConsumerConf.Group)
	}
	serviceGroup.Start()
}
