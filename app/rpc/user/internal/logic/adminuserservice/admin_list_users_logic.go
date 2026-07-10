package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminListUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAdminListUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListUsersLogic {
	return &AdminListUsersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AdminListUsersLogic) AdminListUsers(in *user.AdminListUsersReq) (*user.AdminListUsersRes, error) {
	// todo: add your logic here and delete this line

	return &user.AdminListUsersRes{}, nil
}
