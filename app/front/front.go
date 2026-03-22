// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"
	"ran-feed/pkg/envx"
	"ran-feed/pkg/result"
	"ran-feed/pkg/validate"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

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
