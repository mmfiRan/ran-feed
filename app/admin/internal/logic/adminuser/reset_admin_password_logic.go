// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package adminuser

import (
	"context"

	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResetAdminPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResetAdminPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetAdminPasswordLogic {
	return &ResetAdminPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetAdminPasswordLogic) ResetAdminPassword(req *types.AdminUserResetPasswordReq) (resp *types.AdminUserResetPasswordRes, err error) {
	operatorID := utils.GetContextAdminIdWithDefault(l.ctx)
	_, err = l.svcCtx.AdminRpc.ResetAdminPassword(l.ctx, &admin.ResetAdminPasswordReq{
		Id:          req.Id,
		NewPassword: req.NewPassword,
		OperatorId:  operatorID,
	})
	if err != nil {
		return nil, err
	}

	// 重置密码即踢下线 强制用新密码重登
	kickAdminSession(l.ctx, l.svcCtx, req.Id)
	return &types.AdminUserResetPasswordRes{}, nil
}
