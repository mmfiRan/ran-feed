package main

import (
	"context"
	"flag"
	"fmt"

	"ran-feed/app/rpc/search/internal/config"
	"ran-feed/app/rpc/search/internal/es"
	searchserviceServer "ran-feed/app/rpc/search/internal/server/searchservice"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"
	"ran-feed/pkg/envx"
	"ran-feed/pkg/interceptor"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/search.yaml", "the config file")

func main() {
	flag.Parse()

	envx.Load()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	ctx := svc.NewServiceContext(c)

	// 启动幂等建索引 ES 暂不可用不致命 靠重建 job 兜底
	if err := es.EnsureIndices(context.Background(), ctx.ES); err != nil {
		logx.Errorf("ES 建索引失败 服务继续启动 err=%v", err)
	}

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		search.RegisterSearchServiceServer(grpcServer, searchserviceServer.NewSearchServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	s.AddUnaryInterceptors(interceptor.ServerGrpcInterceptor())
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
