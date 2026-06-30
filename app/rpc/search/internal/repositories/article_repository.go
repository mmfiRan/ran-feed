package repositories

import (
	"context"

	"ran-feed/app/rpc/search/internal/entity/model"
	"ran-feed/app/rpc/search/internal/entity/query"
	"ran-feed/pkg/enum"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type ArticleRepository interface {
	GetByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedArticle, error)
}

type articleRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewArticleRepository(ctx context.Context, db *orm.DB) ArticleRepository {
	return &articleRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// GetByContentIDs 按 content_id 批量取未删除文章 供组装内容文档取标题简介正文
func (r *articleRepositoryImpl) GetByContentIDs(contentIDs []int64) (map[int64]*model.RanFeedArticle, error) {
	res := make(map[int64]*model.RanFeedArticle, len(contentIDs))
	if len(contentIDs) == 0 {
		return res, nil
	}

	q := query.Q
	rows, err := q.RanFeedArticle.WithContext(r.ctx).
		Where(q.RanFeedArticle.IsDeleted.Eq(enum.NotDeleted.Int32())).
		Where(q.RanFeedArticle.ContentID.In(contentIDs...)).
		Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		res[row.ContentID] = row
	}
	return res, nil
}
