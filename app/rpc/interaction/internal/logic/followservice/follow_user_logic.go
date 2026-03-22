package followservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
)

const (
	backfillFollowInboxLimit   = 20
	backfillFollowInboxTimeout = 3 * time.Second
)

type FollowUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	followRepo repositories.FollowRepository
}

func NewFollowUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowUserLogic {
	return &FollowUserLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		followRepo: repositories.NewFollowRepository(ctx, svcCtx.MysqlDb),
	}
}

func (l *FollowUserLogic) FollowUser(in *interaction.FollowUserReq) (*interaction.FollowUserRes, error) {
	if in == nil {
		return &interaction.FollowUserRes{
			IsFollowed: false,
		}, nil
	}
	if in.UserId <= 0 || in.FollowUserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	if in.UserId == in.FollowUserId {
		return nil, errorx.NewMsg("不能关注自己")
	}

	// TODO: 调用 user 服务校验被关注用户是否存在

	err := l.followRepo.Upsert(&do.FollowDO{
		UserID:       in.UserId,
		FollowUserID: in.FollowUserId,
		Status:       repositories.FollowStatusFollow,
		CreatedBy:    in.UserId,
		UpdatedBy:    in.UserId,
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("关注失败"))
	}

	threading.GoSafe(func() {
		_, callErr := l.svcCtx.ContentRpc.BackfillFollowInbox(context.Background(), &content.BackfillFollowInboxReq{
			FollowerId: in.UserId,
			FolloweeId: in.FollowUserId,
			Limit:      backfillFollowInboxLimit,
		})
		if callErr != nil {
			l.Errorf("关注回填收件箱失败: %v", callErr)
		}
	})

	return &interaction.FollowUserRes{
		IsFollowed: true,
	}, nil
}
