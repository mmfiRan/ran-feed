package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/user/internal/common/utils"
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"
	pkgutils "ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("获取用户信息失败"))
	}
	if row == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	detail := &user.AdminUserDetail{
		UserId:    row.ID,
		Username:  row.Username,
		Nickname:  row.Nickname,
		Mobile:    pkgutils.Deref(row.Mobile),
		Avatar:    row.Avatar,
		Status:    utils.UserStatusValue(row.Status),
		Bio:       row.Bio,
		Gender:    utils.GenderValue(row.Gender),
		Email:     pkgutils.Deref(row.Email),
		CreatedAt: timestamppb.New(row.CreatedAt),
		UpdatedAt: timestamppb.New(row.UpdatedAt),
	}

	return &user.AdminGetUserDetailRes{
		Detail: detail,
	}, nil
}
