// 本文件负责关注流读接口的编排 两侧取数在 follow_feed_source 合并在 follow_feed_mapper
package feedservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/contentresolver"
	"ran-feed/app/rpc/content/internal/common/component/followfeed"
	"ran-feed/app/rpc/content/internal/common/component/publishbox"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type FollowFeedLogic struct {
	ctx context.Context
	logx.Logger
	followFeed *followfeed.Feed
	publishBox *publishbox.Box
	resolver   *contentresolver.Resolver
}

func NewFollowFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowFeedLogic {
	return &FollowFeedLogic{
		ctx:        ctx,
		Logger:     logx.WithContext(ctx),
		followFeed: svcCtx.FollowFeed,
		publishBox: svcCtx.PublishBox,
		resolver:   svcCtx.ContentResolver,
	}
}

// FollowFeed 关注流 推侧读收件箱本页 拉侧读拉模式集的作者发件箱 合并去重排序
func (l *FollowFeedLogic) FollowFeed(in *content.FollowFeedReq) (*content.FollowFeedRes, error) {
	if in == nil {
		return emptyFollowFeedRes(), nil
	}
	pageSize := utils.ClampPageSize(in.PageSize)
	cursorScore, cursorID := parseCursor(in.Cursor)

	sources := l.followFeed.Sources(l.ctx, in.UserId, cursorScore, pageSize)
	pullItems, pullHasMore := l.fetchPullItems(sources.PullAuthors, cursorScore, pageSize)

	ids, hasMore, nextCursor := mergeScored(
		followFeedScored(sources.PushItems), sources.PushHasMore,
		pullItems, pullHasMore,
		cursorScore, cursorID, pageSize,
	)
	if len(ids) == 0 {
		return emptyFollowFeedRes(), nil
	}

	// 走统一二级缓存取详情 关注流只读 PUBLIC 再旁挂作者与点赞
	entries, err := l.resolver.Resolve(l.ctx, ids, in.UserId, true)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		// 过滤后为空也返回原始游标 避免整页死内容导致翻页中断
		return &content.FollowFeedRes{Items: []*content.FollowFeedItem{}, NextCursor: nextCursor, HasMore: hasMore}, nil
	}

	return &content.FollowFeedRes{
		Items:      buildFollowItems(entries),
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func emptyFollowFeedRes() *content.FollowFeedRes {
	return &content.FollowFeedRes{
		Items:      []*content.FollowFeedItem{},
		NextCursor: "",
		HasMore:    false,
	}
}
