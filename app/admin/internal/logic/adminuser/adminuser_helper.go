package adminuser

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

// toEnumValue pb 枚举值转 HTTP 响应统一枚举结构
func toEnumValue(v *admin.EnumValue) types.EnumValue {
	if v == nil {
		return types.EnumValue{}
	}
	return types.EnumValue{
		Code:    v.GetCode(),
		Name:    v.GetName(),
		Message: v.GetMessage(),
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
