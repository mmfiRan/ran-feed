package followservicelogic

import (
	"context"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnfollowUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

func NewUnfollowUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnfollowUserLogic {
	return &UnfollowUserLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *UnfollowUserLogic) UnfollowUser(in *interaction.UnfollowUserReq) (*interaction.UnfollowUserRes, error) {
	if in == nil {
		return &interaction.UnfollowUserRes{IsFollowed: false}, nil
	}
	if in.UserId <= 0 || in.FollowUserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.UserId == in.FollowUserId {
		return nil, errorx.NewMsg("不能取关自己")
	}

	// TODO: 调用 user 服务校验被取关用户是否存在

	err := l.followRepo.Upsert(&do.FollowDO{
		UserID:       in.UserId,
		FollowUserID: in.FollowUserId,
		Status:       repositories.FollowStatusUnfollow,
		CreatedBy:    in.UserId,
		UpdatedBy:    in.UserId,
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("取消关注失败"))
	}

	return &interaction.UnfollowUserRes{
		IsFollowed: false,
	}, nil
}
