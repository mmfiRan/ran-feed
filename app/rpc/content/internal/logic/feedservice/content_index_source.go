package feedservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
)

// assembleContentIndexItems 回源 article video 把内容行拼成搜索索引投影 调用方已保证可索引 这里不再过滤
func assembleContentIndexItems(ctx context.Context, articleRepo repositories.ArticleRepository, videoRepo repositories.VideoRepository, contents []*model.RanFeedContent) ([]*content.ContentIndexItem, error) {
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

	articles, err := articleRepo.BatchGetIndexByContentIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	videos, err := videoRepo.BatchGetBriefByContentIDs(ctx, ids)
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
