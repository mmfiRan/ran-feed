package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminSetUserStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAdminSetUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminSetUserStatusLogic {
	return &AdminSetUserStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AdminSetUserStatusLogic) AdminSetUserStatus(in *user.AdminSetUserStatusReq) (*user.AdminSetUserStatusRes, error) {
	// todo: add your logic here and delete this line

	return &user.AdminSetUserStatusRes{}, nil
}
