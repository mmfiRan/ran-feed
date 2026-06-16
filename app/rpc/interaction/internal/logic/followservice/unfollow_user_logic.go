package followservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
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

	// 取关为清理语义：即使被取关用户已被禁/删，也允许 viewer 清理关注关系，故不再校验存在性

	// 读前置状态判断是否真正从关注翻转为取关 读失败默认按翻转处理清理幂等无害
	transitioned := true
	if prior, perr := l.followRepo.GetByUserAndFollow(in.UserId, in.FollowUserId); perr != nil {
		l.Errorf("查关注前置状态失败 默认清理 userID=%d followUserID=%d err=%v", in.UserId, in.FollowUserId, perr)
	} else if prior == nil || prior.Status != repositories.FollowStatusFollow {
		transitioned = false
	}

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

	// 仅真正翻转为取关才清理 收件箱清理与大 V 缓存失效由 content 侧统一处理
	if transitioned {
		threading.GoSafe(func() {
			ctx := context.WithoutCancel(l.ctx)
			_, callErr := l.svcCtx.ContentRpc.PurgeFolloweeFromInbox(ctx, &content.PurgeFolloweeFromInboxReq{
				FollowerId: in.UserId,
				FolloweeId: in.FollowUserId,
			})
			if callErr != nil {
				l.Errorf("取关清理收件箱失败: %v", callErr)
			}
		})
	}

	return &interaction.UnfollowUserRes{
		IsFollowed: false,
	}, nil
}
