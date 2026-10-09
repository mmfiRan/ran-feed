package contentreviewrepo

import (
	"context"

	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/pkg/enums"
)

var _ repositories.ContentReviewRepository = (*Repository)(nil)

type Repository struct {
	tx *query.Query
}

func New() *Repository {
	return &Repository{}
}

func (r *Repository) WithTx(tx *query.Query) repositories.ContentReviewRepository {
	return &Repository{
		tx: tx,
	}
}

func (r *Repository) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *Repository) Create(ctx context.Context, review *model.RanFeedContentReview) error {
	q := r.getQuery()
	return q.RanFeedContentReview.WithContext(ctx).Create(review)
}

func (r *Repository) LatestReasonByContentIDs(ctx context.Context, contentIDs []int64, decisions []contentEnum.ReviewDecisionEnum) (map[int64]string, error) {
	res := make(map[int64]string)
	if len(contentIDs) == 0 || len(decisions) == 0 {
		return res, nil
	}
	q := r.getQuery()
	codes := make([]int32, 0, len(decisions))
	for _, d := range decisions {
		codes = append(codes, d.Int32())
	}
	rows, err := q.RanFeedContentReview.WithContext(ctx).
		Select(q.RanFeedContentReview.ContentID, q.RanFeedContentReview.Reason, q.RanFeedContentReview.CreatedAt).
		Where(q.RanFeedContentReview.ContentID.In(contentIDs...)).
		Where(q.RanFeedContentReview.Decision.In(codes...)).
		Where(q.RanFeedContentReview.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Order(q.RanFeedContentReview.CreatedAt.Desc()).
		Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		if _, ok := res[row.ContentID]; !ok {
			res[row.ContentID] = row.Reason
		}
	}
	return res, nil
}
