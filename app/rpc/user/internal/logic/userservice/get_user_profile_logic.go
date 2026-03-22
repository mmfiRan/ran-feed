package userservicelogic

import (
	"context"
	"ran-feed/app/rpc/user/internal/repositories"
	"ran-feed/app/rpc/user/internal/svc"
	"ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	userRepo repositories.UserRepository
}

func NewGetUserProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserProfileLogic {
	return &GetUserProfileLogic{
		ctx:      ctx,
		svcCtx:   svcCtx,
		Logger:   logx.WithContext(ctx),
		userRepo: repositories.NewUserRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetUserProfileLogic) GetUserProfile(in *user.GetUserProfileReq) (*user.GetUserProfileRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	u, err := l.userRepo.GetByID(in.UserId)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询用户失败"))
	}
	if u == nil {
		return nil, errorx.NewMsg("用户不存在")
	}

	return &user.GetUserProfileRes{
		UserProfile: &user.UserProfile{
			UserId:   u.ID,
			Nickname: u.Nickname,
			Avatar:   u.Avatar,
			Bio:      u.Bio,
			Gender:   user.Gender(u.Gender),
			Status:   user.UserStatus(u.Status),
		},
	}, nil
}
