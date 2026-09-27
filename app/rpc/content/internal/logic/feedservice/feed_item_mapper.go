package feedservicelogic

import (
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/component/contentresolver"
	"ran-feed/app/rpc/content/internal/common/convert"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// buildContentItems 把解析结果组装成 ContentItem 顺序照 entries
func buildContentItems(entries []*contentresolver.Entry) []*content.ContentItem {
	items := make([]*content.ContentItem, 0, len(entries))
	for _, e := range entries {
		d := e.Detail
		items = append(items, &content.ContentItem{
			ContentId:    d.ContentID,
			ContentType:  convert.ContentTypeValue(d.ContentType),
			AuthorId:     d.AuthorID,
			AuthorName:   e.AuthorName,
			AuthorAvatar: e.AuthorAvatar,
			Title:        d.Title,
			CoverUrl:     d.CoverURL,
			PublishedAt:  timestamppb.New(time.Unix(d.PublishedAt, 0)),
			IsLiked:      e.IsLiked,
			LikeCount:    e.LikeCount,
		})
	}
	return items
}

// buildFollowItems 把解析结果组装成 FollowFeedItem 顺序照 entries
func buildFollowItems(entries []*contentresolver.Entry) []*content.FollowFeedItem {
	items := make([]*content.FollowFeedItem, 0, len(entries))
	for _, e := range entries {
		d := e.Detail
		items = append(items, &content.FollowFeedItem{
			ContentId:    d.ContentID,
			ContentType:  convert.ContentTypeValue(d.ContentType),
			AuthorId:     d.AuthorID,
			AuthorName:   e.AuthorName,
			AuthorAvatar: e.AuthorAvatar,
			Title:        d.Title,
			CoverUrl:     d.CoverURL,
			PublishedAt:  timestamppb.New(time.Unix(d.PublishedAt, 0)),
			IsLiked:      e.IsLiked,
			LikeCount:    e.LikeCount,
		})
	}
	return items
}
