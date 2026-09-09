// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.AdminLoginReq) (resp *types.AdminLoginRes, err error) {

	loginRes, err := l.svcCtx.AdminAuthRpc.Login(l.ctx, &admin.LoginReq{
		Username: *req.Username,
		Password: *req.Password,
	})
	if err != nil {
		return nil, err
	}

	return &types.AdminLoginRes{
		AdminId:     loginRes.GetAdminId(),
		Token:       loginRes.GetToken(),
		ExpiredAt:   loginRes.GetTtlSeconds(),
		Nickname:    loginRes.GetNickname(),
		Permissions: loginRes.GetPermissions(),
	}, nil
}
