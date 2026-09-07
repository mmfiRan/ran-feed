package adminuserservicelogic

import (
	"context"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type UpdateAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	adminUserRepo repositories.AdminUserRepository
}

func NewUpdateAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAdminLogic {
	return &UpdateAdminLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		adminUserRepo: repositories.NewAdminUserRepository(ctx, svcCtx.MysqlDb),
	}
}

// UpdateAdmin 改管理员昵称
func (l *UpdateAdminLogic) UpdateAdmin(in *admin.UpdateAdminReq) (*emptypb.Empty, error) {
	nickname := in.GetNickname()
	affected, err := l.adminUserRepo.UpdateProfile(in.GetId(), nickname, in.GetOperatorId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新管理员失败"))
	}
	if affected == 0 {
		return nil, errorx.NewMsg("管理员不存在")
	}
	return &emptypb.Empty{}, nil
}
