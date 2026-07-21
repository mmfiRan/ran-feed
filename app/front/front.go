// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"context"
	"flag"
	"fmt"
	"ran-feed/pkg/envx"
	"ran-feed/pkg/result"
	"ran-feed/pkg/validate"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"ran-feed/app/front/internal/common/sse"
	"ran-feed/app/front/internal/config"
	"ran-feed/app/front/internal/handler"
	"ran-feed/app/front/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/front-api.yaml", "the config file")

func main() {
	flag.Parse()
	envx.Load()
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	server := rest.MustNewServer(c.RestConf, rest.WithCors())
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// 启动 Redis Pub/Sub 订阅 分发 notify:push 到 ConnManager
	// SSE 路由与 rest.WithSSE() 由 notification.api 声明 goctl 生成到 routes.go
	pubsubCtx, cancelPubSub := context.WithCancel(context.Background())
	defer cancelPubSub()
	sse.NewPubSub(c.RedisConfig, ctx.NotifyConnManager).Start(pubsubCtx)

	validator, err := validate.NewCustomValidator()
	if err != nil {
		logx.Errorf("初始化自定义验证器失败: %v", err)
		return
	}
	httpx.SetValidator(validator)
	httpx.SetOkHandler(result.SetCustomSuccessResult)
	httpx.SetErrorHandlerCtx(result.SetCustomErrorResult)
	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
