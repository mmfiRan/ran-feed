package cron

import (
	"ran-feed/app/rpc/content/internal/cron/hotupdate"
	"ran-feed/app/rpc/content/internal/cron/outboxreconcile"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/xxljob"
)

// Register 注册所有内容域的定时任务
func Register(executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	hotupdate.Register(executor, svcCtx)
	outboxreconcile.Register(executor, svcCtx)
}
