package feedservicelogic

import (
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/model"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// publishedMillis 发布时间转毫秒 空时间返回 0
func publishedMillis(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}

// buildContentIndexItem 单条映射 文章带标题 简介 正文 视频只有标题 子表缺失留空不拖垮整批索引
func buildContentIndexItem(c *model.RanFeedContent, article *model.RanFeedArticle, video *model.RanFeedVideo) *content.ContentIndexItem {
	item := &content.ContentIndexItem{
		ContentId:   c.ID,
		ContentType: content.ContentType(c.ContentType),
		Status:      content.ContentStatus(c.Status),
		Visibility:  content.Visibility(c.Visibility),
		AuthorId:    c.UserID,
		PublishedAt: timestamppb.New(time.UnixMilli(publishedMillis(c.PublishedAt))),
		HotScore:    c.HotScore,
		Version:     c.UpdatedAt.UnixMilli(),
	}
	switch c.ContentType {
	case int32(content.ContentType_CONTENT_TYPE_ARTICLE):
		if article != nil {
			item.Title = article.Title
			if article.Description != nil {
				item.Description = *article.Description
			}
			item.Body = article.Content
		}
	case int32(content.ContentType_CONTENT_TYPE_VIDEO):
		if video != nil {
			item.Title = video.Title
		}
	}
	return item
}
