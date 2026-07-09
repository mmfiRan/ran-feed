// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package main

import (
	"flag"
	"fmt"
	"ran-feed/pkg/envx"
	"ran-feed/pkg/result"
	"ran-feed/pkg/validate"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"ran-feed/app/admin/internal/config"
	"ran-feed/app/admin/internal/docmeta"
	"ran-feed/app/admin/internal/handler"
	"ran-feed/app/admin/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/admin-api.yaml", "the config file")

func main() {
	flag.Parse()
	envx.Load()
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	server := rest.MustNewServer(c.RestConf, rest.WithCors())
	defer server.Stop()

	// 全局最先注入 把当前路由 @doc 元数据(含 permission)放进 ctx 供 RBAC 中间件读取
	server.Use(docmeta.Inject)

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
