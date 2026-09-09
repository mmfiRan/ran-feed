// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	pkgconsts "ran-feed/pkg/consts"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutLogic) Logout() (resp *types.AdminLogoutRes, err error) {
	adminID := utils.GetContextAdminIdWithDefault(l.ctx)
	token, _ := l.ctx.Value(pkgconsts.CtxKeyToken).(string)

	keys := make([]string, 0, 2)
	if token != "" {
		keys = append(keys, consts.BuildAdminSessionKey(token))
	}
	if adminID > 0 {
		keys = append(keys, consts.BuildAdminSessionUIDKey(adminID))
	}
	if len(keys) > 0 {
		if _, err = l.svcCtx.Redis.DelCtx(l.ctx, keys...); err != nil {
			return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("删除登录态失败"))
		}
	}

	return &types.AdminLogoutRes{}, nil
}
