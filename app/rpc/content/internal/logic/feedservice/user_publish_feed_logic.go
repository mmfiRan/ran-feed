package feedservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/logic/publishbox"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserPublishFeedLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	resolver   *contentDetailResolver
	publishBox *publishbox.PublishBox
}

func NewUserPublishFeedLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserPublishFeedLogic {
	return &UserPublishFeedLogic{
		ctx:        ctx,
		svcCtx:     svcCtx,
		Logger:     logx.WithContext(ctx),
		resolver:   newContentDetailResolver(ctx, svcCtx),
		publishBox: publishbox.New(ctx, svcCtx),
	}
}

func (l *UserPublishFeedLogic) UserPublishFeed(in *content.UserPublishFeedReq) (*content.UserPublishFeedRes, error) {
	if in == nil {
		return emptyUserPublishFeedRes(), nil
	}
	if in.AuthorId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	pageSize := int(in.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}

	cursorScore, cursorID := parseCursor(in.Cursor)

	// 作者发件箱全量历史 cutoff=0 复用 publishbox 命中读/未命中 PUBLIC-only 重建/空哨兵
	items, hasMore, err := l.publishBox.QueryWindow(in.AuthorId, 0, cursorScore, pageSize)
	if err != nil {
		return nil, err
	}

	// 复合游标 score:id 精确过滤边界同分已展示项
	ids := make([]int64, 0, len(items))
	var last scoredID
	for _, it := range items {
		s := scoredID{id: it.ID, score: it.Score}
		if !afterCursor(s, cursorScore, cursorID) {
			continue
		}
		ids = append(ids, s.id)
		last = s
	}
	if len(ids) == 0 {
		return emptyUserPublishFeedRes(), nil
	}

	viewerID := int64(0)
	if in.ViewerId != nil {
		viewerID = *in.ViewerId
	}
	// PUBLIC-only 与写扩散/大V merge 口径一致 杜绝作者主页把私密内容泄露给任意访问者
	feedItems, err := l.resolver.assembleItems(ids, viewerID, true)
	if err != nil {
		return nil, err
	}
	if len(feedItems) == 0 {
		return emptyUserPublishFeedRes(), nil
	}

	nextCursor := ""
	if hasMore {
		nextCursor = formatCursor(last.score, last.id)
	}

	return &content.UserPublishFeedRes{
		Items:      feedItems,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func emptyUserPublishFeedRes() *content.UserPublishFeedRes {
	return &content.UserPublishFeedRes{
		Items:      []*content.ContentItem{},
		NextCursor: "",
		HasMore:    false,
	}
}