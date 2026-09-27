package repositories

import (
	"context"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
)

type ArticleRepository interface {
	WithTx(tx *query.Query) ArticleRepository
	CreateArticle(ctx context.Context, article *model.RanFeedArticle) error
	UpdateByContentID(ctx context.Context, article *model.RanFeedArticle) error
	DeleteByContentID(ctx context.Context, contentID int64) error
	GetByContentID(ctx context.Context, contentID int64) (*model.RanFeedArticle, error)
	BatchGetBriefByContentIDs(ctx context.Context, contentIDs []int64) (map[int64]*model.RanFeedArticle, error)
	BatchGetIndexByContentIDs(ctx context.Context, contentIDs []int64) (map[int64]*model.RanFeedArticle, error)
}
