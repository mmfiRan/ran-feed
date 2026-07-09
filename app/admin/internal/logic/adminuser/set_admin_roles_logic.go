// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package adminuser

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/common/rbac"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetAdminRolesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetAdminRolesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAdminRolesLogic {
	return &SetAdminRolesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetAdminRolesLogic) SetAdminRoles(req *types.AdminUserSetRolesReq) (resp *types.AdminUserSetRolesRes, err error) {
	operatorID, _ := l.ctx.Value(consts.CtxKeyAdminID).(int64)
	_, err = l.svcCtx.AdminRpc.SetAdminRoles(l.ctx, &admin.SetAdminRolesReq{
		AdminId:    req.AdminId,
		RoleIds:    req.RoleIds,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}

	// 该管理员角色变更 失效其权限缓存使新角色权限立即生效
	if e := rbac.Invalidate(l.ctx, l.svcCtx.Redis, req.AdminId); e != nil {
		logx.WithContext(l.ctx).Errorf("失效管理员权限缓存失败 adminID=%d err=%v", req.AdminId, e)
	}
	return &types.AdminUserSetRolesRes{}, nil
}
