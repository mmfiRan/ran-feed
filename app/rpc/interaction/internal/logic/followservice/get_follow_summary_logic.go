package followservicelogic

import (
	"context"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/mr"
)

type GetFollowSummaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

func NewGetFollowSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFollowSummaryLogic {
	return &GetFollowSummaryLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *GetFollowSummaryLogic) GetFollowSummary(in *interaction.GetFollowSummaryReq) (*interaction.GetFollowSummaryRes, error) {
	if in == nil {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	var (
		followeeCount int64
		followerCount int64
		isFollowing   bool
	)

	err := mr.Finish(
		func() error {
			n, e := l.followRepo.CountFollowees(in.UserId)
			if e != nil {
				return e
			}
			followeeCount = n
			return nil
		},
		func() error {
			n, e := l.followRepo.CountFollowers(in.UserId)
			if e != nil {
				return e
			}
			followerCount = n
			return nil
		},
		func() error {
			viewerID := in.GetViewerId()
			if viewerID <= 0 {
				isFollowing = false
				return nil
			}
			b, e := l.followRepo.IsFollowing(viewerID, in.UserId)
			if e != nil {
				return e
			}
			isFollowing = b
			return nil
		},
	)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询关注信息失败"))
	}

	return &interaction.GetFollowSummaryRes{
		FolloweeCount: followeeCount,
		FollowerCount: followerCount,
		IsFollowing:   isFollowing,
	}, nil
}
