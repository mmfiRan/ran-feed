package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetUserDetailLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAdminGetUserDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetUserDetailLogic {
	return &AdminGetUserDetailLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AdminGetUserDetailLogic) AdminGetUserDetail(in *user.AdminGetUserDetailReq) (*user.AdminGetUserDetailRes, error) {
	// todo: add your logic here and delete this line

	return &user.AdminGetUserDetailRes{}, nil
}
