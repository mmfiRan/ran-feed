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

// backfillWindowCap 新关注回填的窗口内条数上限
const backfillWindowCap = 200

type BackfillFollowInboxLogic struct {
	ctx context.Context
	logx.Logger
	followFeed *followfeed.Feed
	publishBox *publishbox.Box
	bigv       *bigv.Set
}

func NewBackfillFollowInboxLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BackfillFollowInboxLogic {
	return &BackfillFollowInboxLogic{
		ctx:        ctx,
		Logger:     logx.WithContext(ctx),
		followFeed: svcCtx.FollowFeed,
		publishBox: svcCtx.PublishBox,
		bigv:       svcCtx.BigV,
	}
}

// BackfillFollowInbox 新关注后把 followee 窗口内已发布内容灌进 follower 收件箱
func (l *BackfillFollowInboxLogic) BackfillFollowInbox(in *content.BackfillFollowInboxReq) (*content.BackfillFollowInboxRes, error) {
	if in == nil {
		return &content.BackfillFollowInboxRes{AddedCount: 0}, nil
	}
	if in.FollowerId <= 0 || in.FolloweeId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}

	// 关注关系变更 失效 viewer 拉模式集 下次读按新关注集合重算
	if err := l.followFeed.InvalidatePull(l.ctx, in.FollowerId); err != nil {
		l.Errorf("失效拉模式关注集失败 viewerID=%d err=%v", in.FollowerId, err)
	}

	// 拉模式作者不回填 其内容由读路径查其发件箱覆盖 与写扩散对称 查询失败保守仍回填
	if isBig, berr := l.bigv.IsBigV(l.ctx, in.FolloweeId); berr != nil {
		l.Errorf("查询大 V 集合失败 followeeID=%d err=%v", in.FolloweeId, berr)
	} else if isBig {
		return &content.BackfillFollowInboxRes{AddedCount: 0}, nil
	}

	limit := int(in.Limit)
	if limit <= 0 || limit > backfillWindowCap {
		limit = backfillWindowCap
	}
	items, err := l.publishBox.ListWindow(l.ctx, in.FolloweeId, contentconsts.WindowCutoffMillis(), limit)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return &content.BackfillFollowInboxRes{AddedCount: 0}, nil
	}

	added, err := l.followFeed.Backfill(l.ctx, in.FollowerId, toInboxItems(items))
	if err != nil {
		return nil, err
	}
	return &content.BackfillFollowInboxRes{AddedCount: int32(added)}, nil
}

// toInboxItems 发件箱候选转收件箱成员 两侧都是 content_id 加发布时间毫秒
func toInboxItems(items []publishbox.ScoredID) []followfeed.ScoredID {
	out := make([]followfeed.ScoredID, 0, len(items))
	for _, it := range items {
		out = append(out, followfeed.ScoredID{ID: it.ID, Score: it.Score})
	}
	return out
}
