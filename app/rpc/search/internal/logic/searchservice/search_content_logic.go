package searchservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/search/internal/svc"
	"ran-feed/app/rpc/search/search"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchContentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchContentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchContentLogic {
	return &SearchContentLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SearchContentLogic) SearchContent(in *search.SearchContentReq) (*search.SearchContentRes, error) {
	if in == nil || in.Keyword == "" {
		return &search.SearchContentRes{}, nil
	}

	size := normalizeSize(in.Size)
	searchAfter, err := decodeCursor(in.Cursor)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("内容搜索失败"))
	}

	result, err := es.Search(l.ctx, l.svcCtx.ES, es.IndexContent, l.buildQuery(in, size, searchAfter))
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("内容搜索失败"))
	}

	hits := make([]*search.ContentHit, 0, len(result.Hits))
	for _, h := range result.Hits {
		contentID, parseErr := strconv.ParseInt(h.ID, 10, 64)
		if parseErr != nil {
			l.Errorf("内容命中 id 解析失败 id=%s err=%v", h.ID, parseErr)
			continue
		}
		hits = append(hits, &search.ContentHit{
			ContentId:            contentID,
			Score:                h.Score,
			HighlightTitle:       h.FirstHighlight("title"),
			HighlightDescription: h.FirstHighlight("description"),
		})
	}
	return &search.SearchContentRes{
		Hits:       hits,
		Total:      result.Total,
		NextCursor: nextCursor(result.Hits, size),
	}, nil
}

// buildQuery 标题^3 简介^2 正文 多字段匹配 仅已发布公开未删除 排序 _score 加 hot_score 加 published_at 末位 content_id 兜底唯一序
// searchAfter 非空时接 search_after 游标翻页 不用 from 偏移
func (l *SearchContentLogic) buildQuery(in *search.SearchContentReq, size int, searchAfter []any) map[string]any {
	filters := []map[string]any{
		{"term": map[string]any{"status": int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED)}},
		{"term": map[string]any{"visibility": int32(content.Visibility_VISIBILITY_PUBLIC)}},
		{"term": map[string]any{"is_deleted": 0}},
	}
	if in.ContentType != search.ContentType_CONTENT_TYPE_UNKNOWN {
		filters = append(filters, map[string]any{"term": map[string]any{"content_type": int32(in.ContentType)}})
	}

	query := map[string]any{
		"size": size,
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{"multi_match": map[string]any{
						"query":    in.Keyword,
						"fields":   []string{"title^3", "description^2", "body"},
						"analyzer": "ik_smart",
					}},
				},
				"filter": filters,
			},
		},
		"sort": []map[string]any{
			{"_score": map[string]any{"order": "desc"}},
			{"hot_score": map[string]any{"order": "desc"}},
			{"published_at": map[string]any{"order": "desc"}},
			{"content_id": map[string]any{"order": "asc"}},
		},
		"highlight": map[string]any{
			"fields": map[string]any{
				"title":       map[string]any{},
				"description": map[string]any{},
			},
		},
	}
	if len(searchAfter) > 0 {
		query["search_after"] = searchAfter
	}
	return query
}
