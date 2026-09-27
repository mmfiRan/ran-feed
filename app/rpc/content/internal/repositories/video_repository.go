package repositories

import (
	"context"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
)

type VideoRepository interface {
	WithTx(tx *query.Query) VideoRepository
	CreateVideo(ctx context.Context, video *model.RanFeedVideo) error
	UpdateByContentID(ctx context.Context, video *model.RanFeedVideo) error
	DeleteByContentID(ctx context.Context, contentID int64) error
	GetByContentID(ctx context.Context, contentID int64) (*model.RanFeedVideo, error)
	BatchGetBriefByContentIDs(ctx context.Context, contentIDs []int64) (map[int64]*model.RanFeedVideo, error)
}
