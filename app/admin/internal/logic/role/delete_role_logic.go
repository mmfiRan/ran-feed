// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package role

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/pkg/utils"

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
	operatorID := utils.GetContextAdminIdWithDefault(l.ctx)
	_, err = l.svcCtx.AdminRoleRpc.DeleteRole(l.ctx, &admin.DeleteRoleReq{
		Id:         req.Id,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}
	return &types.AdminRoleDeleteRes{}, nil
}
