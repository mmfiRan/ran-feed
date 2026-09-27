// 本文件负责关注流读接口的编排 取数在 follow_feed_source 缓存维护在 follow_feed_cache
package feedservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/contentresolver"
	"ran-feed/app/rpc/content/internal/common/component/publishbox"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/app/rpc/interaction/client/followservice"
	"ran-feed/pkg/cache"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type FollowFeedLogic struct {
	ctx context.Context
	logx.Logger
	redis               *redis.Redis
	followRebuildLocker *cache.DistLocker
	followRpc           followservice.FollowService
	contentRepo         repositories.ContentRepository
	resolver            *contentresolver.Resolver
	publishBox          *publishbox.Box
}

func NewFollowFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FollowFeedLogic {
	return &FollowFeedLogic{
		ctx:                 ctx,
		Logger:              logx.WithContext(ctx),
		redis:               svcCtx.Redis,
		followRebuildLocker: svcCtx.FollowRebuildLocker,
		followRpc:           svcCtx.FollowRpc,
		contentRepo:         svcCtx.ContentRepository,
		resolver:            svcCtx.ContentResolver,
		publishBox:          svcCtx.PublishBox,
	}
}

// FollowFeed 关注流 推侧读收件箱本页 拉侧读拉模式集的作者发件箱 合并去重排序
func (l *FollowFeedLogic) FollowFeed(in *content.FollowFeedReq) (*content.FollowFeedRes, error) {
	if in == nil {
		return emptyFollowFeedRes(), nil
	}
	userID := in.UserId
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	cursorScore, cursorID := parseCursor(in.Cursor)

	pushItems, pushHasMore, pullAuthors := l.loadFollowSources(userID, cursorScore, cursorID, pageSize)

	var pool []scoredID
	var poolHasMore bool
	if len(pullAuthors) > 0 {
		pool, poolHasMore = l.fetchPullContentIDs(pullAuthors, cursorScore, pageSize)
	}
	ids, hasMore, nextCursor := mergeScored(pushItems, pushHasMore, pool, poolHasMore, cursorScore, cursorID, pageSize)

	if len(ids) == 0 {
		return emptyFollowFeedRes(), nil
	}

	// 走统一二级缓存取详情 关注流只读 PUBLIC 再旁挂作者与点赞
	entries, err := l.resolver.Resolve(l.ctx, ids, userID, true)
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
