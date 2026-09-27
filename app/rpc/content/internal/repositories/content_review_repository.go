package repositories

import (
	"context"

	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
)

type ContentReviewRepository interface {
	WithTx(tx *query.Query) ContentReviewRepository
	Create(ctx context.Context, review *model.RanFeedContentReview) error
	// LatestReasonByContentIDs 批量取内容最新一条指定决策的理由
	LatestReasonByContentIDs(ctx context.Context, contentIDs []int64, decisions []contentEnum.ReviewDecisionEnum) (map[int64]string, error)
}
