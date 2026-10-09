package feedservicelogic

import (
	"time"

	"ran-feed/app/rpc/content/content"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
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
	// 这三个字段是构造给 search 域的协议消息 属 RPC 边界 故直接落 pb 枚举 不转业务枚举
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
	switch contentEnum.ContentTypeEnum(c.ContentType) {
	case contentEnum.ContentTypeArticle:
		if article != nil {
			item.Title = article.Title
			if article.Description != nil {
				item.Description = *article.Description
			}
			item.Body = article.Content
		}
	case contentEnum.ContentTypeVideo:
		if video != nil {
			item.Title = video.Title
		}
	}
	return item
}
