// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package role

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteRoleLogic {
	return &DeleteRoleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteRoleLogic) DeleteRole(req *types.AdminRoleDeleteReq) (resp *types.AdminRoleDeleteRes, err error) {
	operatorID, _ := l.ctx.Value(consts.CtxKeyAdminID).(int64)
	rpcRes, err := l.svcCtx.AdminRpc.DeleteRole(l.ctx, &admin.DeleteRoleReq{
		Id:         req.Id,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}

	// 角色删除 失效持有该角色的管理员权限缓存
	invalidatePerms(l.ctx, l.svcCtx, rpcRes.GetAffectedAdminIds())
	return &types.AdminRoleDeleteRes{}, nil
}
