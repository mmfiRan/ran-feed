package cron

import (
	"context"
	"ran-feed/app/rpc/content/internal/cron/hot_update"
	"ran-feed/app/rpc/content/internal/cron/outbox_reconcile"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/xxljob"
)

// Register 注册所有内容域的定时任务。
func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	hot_update.Register(ctx, executor, svcCtx)
	outbox_reconcile.Register(ctx, executor, svcCtx)
}
