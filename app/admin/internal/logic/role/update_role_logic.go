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

type UpdateRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoleLogic {
	return &UpdateRoleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateRoleLogic) UpdateRole(req *types.AdminRoleUpdateReq) (resp *types.AdminRoleUpdateRes, err error) {
	operatorID, _ := l.ctx.Value(consts.CtxKeyAdminID).(int64)
	_, err = l.svcCtx.AdminRpc.UpdateRole(l.ctx, &admin.UpdateRoleReq{
		Id:         req.Id,
		Name:       req.Name,
		Remark:     req.Remark,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}
	return &types.AdminRoleUpdateRes{}, nil
}
