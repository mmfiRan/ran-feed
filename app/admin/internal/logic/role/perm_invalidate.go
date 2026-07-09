package role

import (
	"context"

	"ran-feed/app/admin/internal/common/rbac"
	"ran-feed/app/admin/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// invalidatePerms 遍历失效受影响管理员的权限缓存 失败只 log 不阻断
func invalidatePerms(ctx context.Context, svcCtx *svc.ServiceContext, adminIDs []int64) {
	for _, id := range adminIDs {
		if id <= 0 {
			continue
		}
		if err := rbac.Invalidate(ctx, svcCtx.Redis, id); err != nil {
			logx.WithContext(ctx).Errorf("失效管理员权限缓存失败 adminID=%d err=%v", id, err)
		}
	}
}
