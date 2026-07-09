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

type CreateAdminLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAdminLogic {
	return &CreateAdminLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateAdminLogic) CreateAdmin(req *types.AdminUserCreateReq) (resp *types.AdminUserCreateRes, err error) {
	operatorID, _ := l.ctx.Value(consts.CtxKeyAdminID).(int64)
	rpcRes, err := l.svcCtx.AdminRpc.CreateAdmin(l.ctx, &admin.CreateAdminReq{
		Username:   req.Username,
		Password:   req.Password,
		Nickname:   req.Nickname,
		RoleIds:    req.RoleIds,
		OperatorId: operatorID,
	})
	if err != nil {
		return nil, err
	}
	return &types.AdminUserCreateRes{Id: rpcRes.GetId()}, nil
}
