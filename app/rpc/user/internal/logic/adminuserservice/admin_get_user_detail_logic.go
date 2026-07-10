package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetUserDetailLogic struct {
	ctx      context.Context
	svcCtx   *svc.ServiceContext
	userRepo repositories.UserRepository
	logx.Logger
}

func NewAdminGetUserDetailLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetUserDetailLogic {
	return &AdminGetUserDetailLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *AdminGetUserDetailLogic) AdminGetUserDetail(in *user.AdminGetUserDetailReq) (*user.AdminGetUserDetailRes, error) {
	row, err := l.userRepo.AdminGetByID(in.UserId)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	detail := &user.AdminUserDetail{
		UserId:    row.ID,
		Username:  row.Username,
		Nickname:  row.Nickname,
		Mobile:    row.Mobile,
		Avatar:    row.Avatar,
		Status:    user.UserStatus(row.Status),
		Bio:       row.Bio,
		Gender:    user.Gender(row.Gender),
		Email:     row.Email,
		CreatedAt: row.CreatedAt.UnixMilli(),
		UpdatedAt: row.UpdatedAt.UnixMilli(),
	}

	return &user.AdminGetUserDetailRes{Detail: detail}, nil
}
