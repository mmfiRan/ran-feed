package adminuser

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

// mapStatusAction 启禁 action 映射状态枚举 第二返回 false 表示不支持
func mapStatusAction(action string) (admin.AdminStatus, bool) {
	switch action {
	case consts.AdminUserActionEnable:
		return admin.AdminStatus_ADMIN_ENABLED, true
	case consts.AdminUserActionDisable:
		return admin.AdminStatus_ADMIN_DISABLED, true
	default:
		return admin.AdminStatus_ADMIN_STATUS_UNKNOWN, false
	}
}

// kickAdminSession 踢管理员下线 读反向索引拿 token 删双向会话 失败只 log
func kickAdminSession(ctx context.Context, svcCtx *svc.ServiceContext, adminID int64) {
	r := svcCtx.Redis
	adminKey := consts.BuildAdminSessionUIDKey(adminID)
	token, err := r.GetCtx(ctx, adminKey)
	if err != nil {
		logx.WithContext(ctx).Errorf("踢下线读会话失败 adminID=%d err=%v", adminID, err)
		return
	}
	if token != "" {
		if _, e := r.DelCtx(ctx, consts.BuildAdminSessionKey(token)); e != nil {
			logx.WithContext(ctx).Errorf("踢下线删 token 会话失败 adminID=%d err=%v", adminID, e)
		}
	}
	if _, e := r.DelCtx(ctx, adminKey); e != nil {
		logx.WithContext(ctx).Errorf("踢下线删反向会话失败 adminID=%d err=%v", adminID, e)
	}
}
