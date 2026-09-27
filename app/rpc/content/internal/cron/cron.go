package cron

import (
	"context"
	"ran-feed/app/rpc/content/internal/cron/hotupdate"
	"ran-feed/app/rpc/content/internal/cron/outboxreconcile"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/xxljob"
)

// Register 注册所有内容域的定时任务
func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	hotupdate.Register(ctx, executor, svcCtx)
	outboxreconcile.Register(ctx, executor, svcCtx)
}
