package cron

import (
	"context"

	"ran-feed/app/rpc/search/internal/cron/search_reindex"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/pkg/xxljob"
)

// Register 注册所有搜索域的定时任务
func Register(ctx context.Context, executor *xxljob.Executor, svcCtx *svc.ServiceContext) {
	search_reindex.Register(ctx, executor, svcCtx)
}
