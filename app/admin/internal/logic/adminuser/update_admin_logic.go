// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package adminuser

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAdminLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAdminLogic {
	return &UpdateAdminLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateAdminLogic) UpdateAdmin(req *types.AdminUserUpdateReq) (resp *types.AdminUserUpdateRes, err error) {
	operatorID, _ := l.ctx.Value(consts.CtxKeyAdminID).(int64)
	_, err = l.svcCtx.AdminRpc.UpdateAdmin(l.ctx, &admin.UpdateAdminReq{
		Id:         req.Id,
		Nickname:   req.Nickname,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}
	return &types.AdminUserUpdateRes{}, nil
}
