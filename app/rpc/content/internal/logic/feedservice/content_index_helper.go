package feedservicelogic

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// publishedMillis 空时间返回 0
func publishedMillis(t *time.Time) int64 {
	if t == nil {
		return 0
	}
	return t.UnixMilli()
}

// assembleContentIndexItems 回源 article/video 组装搜索索引投影 文章取标题简介正文 视频只有标题
// 入参 contents 已保证可索引(已发布+公开+未删除) 不做二次过滤
func assembleContentIndexItems(ctx context.Context, svcCtx *svc.ServiceContext, contents []*model.RanFeedContent) ([]*content.ContentIndexItem, error) {
	items := make([]*content.ContentIndexItem, 0, len(contents))
	if len(contents) == 0 {
		return items, nil
	}

	ids := make([]int64, 0, len(contents))
	for _, c := range contents {
		if c != nil {
			ids = append(ids, c.ID)
		}
	}

	articleRepo := repositories.NewArticleRepository(ctx, svcCtx.MysqlDb)
	videoRepo := repositories.NewVideoRepository(ctx, svcCtx.MysqlDb)
	articles, err := articleRepo.BatchGetIndexByContentIDs(ids)
	if err != nil {
		return nil, err
	}
	videos, err := videoRepo.BatchGetBriefByContentIDs(ids)
	if err != nil {
		return nil, err
	}

	for _, c := range contents {
		if c == nil {
			continue
		}
		items = append(items, buildContentIndexItem(c, articles[c.ID], videos[c.ID]))
	}
	return items, nil
}

// buildContentIndexItem 单条映射 文章取标题简介正文 视频只有标题 子表缺失则相应字段空
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
