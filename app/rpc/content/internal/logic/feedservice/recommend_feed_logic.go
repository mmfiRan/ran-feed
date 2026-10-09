package feedservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/contentresolver"
	"ran-feed/app/rpc/content/internal/common/component/hotfeed"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type RecommendFeedLogic struct {
	ctx context.Context
	logx.Logger
	hotFeed     *hotfeed.Feed
	resolver    *contentresolver.Resolver
	contentRepo repositories.ContentRepository
}

func NewRecommendFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecommendFeedLogic {
	return &RecommendFeedLogic{
		ctx:         ctx,
		Logger:      logx.WithContext(ctx),
		hotFeed:     svcCtx.HotFeed,
		resolver:    svcCtx.ContentResolver,
		contentRepo: svcCtx.ContentRepository,
	}
}

func (l *RecommendFeedLogic) RecommendFeed(in *content.RecommendFeedReq) (*content.RecommendFeedRes, error) {
	pageSize := utils.ClampPageSize(in.PageSize)

	page, err := l.queryPage(in.GetSnapshotId(), in.Cursor, pageSize)
	if err != nil {
		return nil, err
	}
	if len(page.ContentIDs) == 0 {
		return &content.RecommendFeedRes{
			Items:      []*content.ContentItem{},
			NextCursor: "",
			HasMore:    false,
			SnapshotId: page.ResolvedSnapshotID,
		}, nil
	}

	userID := int64(0)
	if in.UserId != nil {
		userID = *in.UserId
	}
	// 热榜只读 PUBLIC 走统一二级缓存
	entries, err := l.resolver.Resolve(l.ctx, page.ContentIDs, userID, true)
	if err != nil {
		return nil, err
	}
	items := buildContentItems(entries)
	if len(items) == 0 {
		// 过滤后为空也返回原始游标 避免整页死内容导致翻页中断
		return &content.RecommendFeedRes{
			Items:      nil,
			NextCursor: page.NextCursor,
			HasMore:    page.HasMore,
			SnapshotId: page.ResolvedSnapshotID,
		}, nil
	}

	return &content.RecommendFeedRes{
		Items:      items,
		NextCursor: page.NextCursor,
		HasMore:    page.HasMore,
		SnapshotId: page.ResolvedSnapshotID,
	}, nil
}

// queryPage 先读热榜 读不到按 hot_score 游标查库兜底 保证推荐流不整接口失败
func (l *RecommendFeedLogic) queryPage(snapshotID, cursor string, pageSize int) (hotfeed.Page, error) {
	page, hit, err := l.hotFeed.Query(l.ctx, snapshotID, cursor, pageSize)
	if err != nil {
		l.Errorf("查询热榜失败 转查库兜底 err=%v", err)
		return l.pageFromDB(cursor, pageSize)
	}
	if !hit {
		// Redis 丢数据或主榜快照均缺失
		return l.pageFromDB(cursor, pageSize)
	}
	return page, nil
}

// pageFromDB 兜底查询 已发布加公开加未删 按 hot_score 与 id 倒序 keyset 翻页
func (l *RecommendFeedLogic) pageFromDB(cursor string, pageSize int) (hotfeed.Page, error) {
	cursorID := int64(0)
	if v, err := strconv.ParseInt(cursor, 10, 64); err == nil && v > 0 {
		cursorID = v
	}

	cursorScore := 0.0
	if cursorID > 0 {
		// 取游标内容的分值定位翻页位置 取不到说明该内容已删 退化为首页
		score, err := l.contentRepo.GetHotScoreByID(l.ctx, cursorID)
		if err != nil {
			l.Errorf("兜底查询解析游标分值失败 退化为首页 cursorID=%d err=%v", cursorID, err)
			cursorID, cursorScore = 0, 0
		} else {
			cursorScore = score
		}
	}

	rows, err := l.contentRepo.ListRecommendByHotScoreCursor(l.ctx,
		contentEnum.ContentStatusPublished,
		contentEnum.VisibilityPublic,
		cursorScore,
		cursorID,
		pageSize+1,
	)
	if err != nil {
		return hotfeed.Page{}, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询推荐流失败"))
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		ids = append(ids, row.ID)
	}
	hasMore := len(ids) > pageSize
	if hasMore {
		ids = ids[:pageSize]
	}
	nextCursor := ""
	if hasMore && len(ids) > 0 {
		nextCursor = strconv.FormatInt(ids[len(ids)-1], 10)
	}
	return hotfeed.Page{ContentIDs: ids, NextCursor: nextCursor, HasMore: hasMore}, nil
}
