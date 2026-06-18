package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/config"
	"ran-feed/app/rpc/count/internal/cron"
	"ran-feed/app/rpc/count/internal/mq/consumer"
	counterserviceServer "ran-feed/app/rpc/count/internal/server/counterservice"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/envx"
	"ran-feed/pkg/interceptor"
	"ran-feed/pkg/xxljob"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/threading"
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

	// xxl-job 执行器 大 V 定时修正任务
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
			// xxlCtx 取消属于正常停机路径
			if errors.Is(err, context.Canceled) {
				return
			}
			// 启动失败意味着大 V 定时修正永久不可用 必须退出由 supervisor 拉起
			logx.Errorf("xxl-job executor start failed, exiting for supervisor restart: %v", err)
			os.Exit(1)
		}
	})

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	if c.KqConsumerConf.Topic != "" {
		fmt.Printf("Starting canal mq consumer for topic: %s, group: %s...\n", c.KqConsumerConf.Topic, c.KqConsumerConf.Group)
	}
	serviceGroup.Start()
}
