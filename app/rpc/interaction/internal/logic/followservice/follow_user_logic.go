package followservicelogic

import (
	"context"
	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/app/rpc/user/client/userservice"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
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

	// 校验被关注用户存在
	resp, gerr := l.svcCtx.UserRpc.GetUser(l.ctx, &userservice.GetUserReq{
		UserId: in.FollowUserId,
	})
	if gerr != nil {
		return nil, gerr
	}
	if resp == nil || resp.UserInfo == nil {
		return nil, errorx.NewMsg("被关注用户不存在")
	}

	// 读前置状态判断是否真正翻转为关注 读失败默认按翻转处理回填幂等无害
	transitioned := true
	if prior, perr := l.followRepo.GetByUserAndFollow(in.UserId, in.FollowUserId); perr != nil {
		l.Errorf("查关注前置状态失败 默认回填 userID=%d followUserID=%d err=%v", in.UserId, in.FollowUserId, perr)
	} else if prior != nil && prior.Status == repositories.FollowStatusFollow {
		transitioned = false
	}

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

	// 仅真正翻转为关注才回填 重复关注不触发 大 V 跳过与缓存失效由 content 侧统一处理
	if transitioned {
		threading.GoSafe(func() {
			ctx := context.WithoutCancel(l.ctx)
			// Limit 留 0 由 content 侧按 deadline 窗口与上限决定回填量
			_, callErr := l.svcCtx.ContentRpc.BackfillFollowInbox(ctx, &content.BackfillFollowInboxReq{
				FollowerId: in.UserId,
				FolloweeId: in.FollowUserId,
			})
			if callErr != nil {
				l.Errorf("关注回填收件箱失败: %v", callErr)
			}
		})
	}

	return &interaction.FollowUserRes{
		IsFollowed: true,
	}, nil
}
