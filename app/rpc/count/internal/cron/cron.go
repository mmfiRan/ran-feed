package cron

import (
	"context"

	"ran-feed/app/rpc/count/internal/cron/bigv_reconcile"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/xxljob"
)

// Register 注册所有计数域的定时任务
func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	bigv_reconcile.Register(ctx, executor, svcCtx)
}
