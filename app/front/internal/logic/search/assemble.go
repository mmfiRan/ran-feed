package search

import (
	"ran-feed/app/front/internal/types"
	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/search/search"
)

// assembleSearchContentItems 按 content-rpc 富化后的顺序拼装 并按 content_id 合并搜索高亮
func assembleSearchContentItems(enriched []*content.ContentItem, highlights map[int64]*search.ContentHit) []types.SearchContentItem {
	items := make([]types.SearchContentItem, 0, len(enriched))
	for _, it := range enriched {
		if it == nil {
			continue
		}
		item := types.SearchContentItem{
			ContentId:    it.ContentId,
			ContentType:  int32(it.ContentType),
			AuthorId:     it.AuthorId,
			AuthorName:   it.AuthorName,
			AuthorAvatar: it.AuthorAvatar,
			Title:        it.Title,
			CoverUrl:     it.CoverUrl,
			PublishedAt:  it.PublishedAt,
			IsLiked:      it.IsLiked,
			LikeCount:    it.LikeCount,
		}
		if hl := highlights[it.ContentId]; hl != nil {
			item.HighlightTitle = hl.HighlightTitle
			item.HighlightDescription = hl.HighlightDescription
		}
		items = append(items, item)
	}
	return items
}
