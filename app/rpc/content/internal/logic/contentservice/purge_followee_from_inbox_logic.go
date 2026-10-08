package contentservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/bigv"
	"ran-feed/app/rpc/content/internal/common/component/followfeed"
	"ran-feed/app/rpc/content/internal/common/component/publishbox"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type PurgeFolloweeFromInboxLogic struct {
	ctx context.Context
	logx.Logger
	followFeed *followfeed.Feed
	publishBox *publishbox.Box
	bigv       *bigv.Set
}

func NewPurgeFolloweeFromInboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PurgeFolloweeFromInboxLogic {
	return &PurgeFolloweeFromInboxLogic{
		ctx:        ctx,
		Logger:     logx.WithContext(ctx),
		followFeed: svcCtx.FollowFeed,
		publishBox: svcCtx.PublishBox,
		bigv:       svcCtx.BigV,
	}
}

// PurgeFolloweeFromInbox 取关后从 follower 收件箱清理 followee 窗口内已扩散内容 对称于 BackfillFollowInbox
func (l *PurgeFolloweeFromInboxLogic) PurgeFolloweeFromInbox(in *content.PurgeFolloweeFromInboxReq) (*content.PurgeFolloweeFromInboxRes, error) {
	if in == nil {
		return &content.PurgeFolloweeFromInboxRes{RemovedCount: 0}, nil
	}
	if in.FollowerId <= 0 || in.FolloweeId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	// 关注关系变更 失效 viewer 拉模式集 取关后读路径才会停止拉该作者
	if err := l.followFeed.InvalidatePull(l.ctx, in.FollowerId); err != nil {
		l.Errorf("失效拉模式关注集失败 viewerID=%d err=%v", in.FollowerId, err)
	}

	// 拉模式内容从未推入收件箱 无需清理 查询失败仍继续清理 对拉模式无害对小号必要
	if isBig, err := l.bigv.IsBigV(l.ctx, in.FolloweeId); err != nil {
		l.Errorf("查询大 V 集合失败 followeeID=%d err=%v", in.FolloweeId, err)
	} else if isBig {
		return &content.PurgeFolloweeFromInboxRes{RemovedCount: 0}, nil
	}

	items, err := l.publishBox.ListWindow(l.ctx, in.FolloweeId, contentconsts.WindowCutoffMillis(), int(contentconsts.TimelineKeepN))
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return &content.PurgeFolloweeFromInboxRes{RemovedCount: 0}, nil
	}

	contentIDs := make([]int64, 0, len(items))
	for _, it := range items {
		contentIDs = append(contentIDs, it.ID)
	}
	removed, err := l.followFeed.RemoveContents(l.ctx, in.FollowerId, contentIDs)
	if err != nil {
		return nil, err
	}
	return &content.PurgeFolloweeFromInboxRes{RemovedCount: int32(removed)}, nil
}
