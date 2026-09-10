package utils

import (
	"context"

	"ran-feed/app/rpc/admin/internal/common/consts"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// InvalidateAdminPerms 失效受影响管理员的权限点缓存
func InvalidateAdminPerms(ctx context.Context, r *redis.Redis, adminIDs []int64) {
	keys := make([]string, 0, len(adminIDs))
	for _, id := range adminIDs {
		if id <= 0 {
			continue
		}
		keys = append(keys, consts.BuildAdminPermKey(id))
	}
	if len(keys) == 0 {
		return
	}
	if err := r.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, keys...)
		return nil
	}); err != nil {
		logx.WithContext(ctx).Errorf("失效管理员权限缓存失败 adminIDs=%v err=%v", adminIDs, err)
	}
}

// InvalidateAdminPerm 失效单个管理员的权限点缓存
func InvalidateAdminPerm(ctx context.Context, r *redis.Redis, adminID int64) {
	if adminID <= 0 {
		return
	}
	if _, err := r.DelCtx(ctx, consts.BuildAdminPermKey(adminID)); err != nil {
		logx.WithContext(ctx).Errorf("失效管理员权限缓存失败 adminID=%d err=%v", adminID, err)
	}
}
