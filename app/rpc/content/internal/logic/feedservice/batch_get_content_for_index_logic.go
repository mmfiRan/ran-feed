package feedservicelogic

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchGetContentForIndexLogic struct {
	ctx context.Context
	logx.Logger
	contentRepo repositories.ContentRepository
	articleRepo repositories.ArticleRepository
	videoRepo   repositories.VideoRepository
}

func NewBatchGetContentForIndexLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchGetContentForIndexLogic {
	return &BatchGetContentForIndexLogic{
		ctx:         ctx,
		Logger:      logx.WithContext(ctx),
		contentRepo: svcCtx.ContentRepository,
		articleRepo: svcCtx.ArticleRepository,
		videoRepo:   svcCtx.VideoRepository,
	}
}

// BatchGetContentForIndex 增量回源 只返回可索引内容的原始投影 不可索引的 id 直接缺席由 search 判删
func (l *BatchGetContentForIndexLogic) BatchGetContentForIndex(in *content.BatchGetContentForIndexReq) (*content.BatchGetContentForIndexRes, error) {
	if in == nil || len(in.ContentIds) == 0 {
		return &content.BatchGetContentForIndexRes{Items: []*content.ContentIndexItem{}}, nil
	}

	rows, err := l.contentRepo.BatchGetIndexableByIDs(l.ctx, in.ContentIds)
	if err != nil {
		return nil, err
	}

	contents := make([]*model.RanFeedContent, 0, len(rows))
	for _, id := range in.ContentIds {
		if row := rows[id]; row != nil {
			contents = append(contents, row)
		}
	}

	items, err := assembleContentIndexItems(l.ctx, l.articleRepo, l.videoRepo, contents)
	if err != nil {
		return nil, err
	}
	return &content.BatchGetContentForIndexRes{Items: items}, nil
}
