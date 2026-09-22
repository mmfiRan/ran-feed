package repositories

import (
	"context"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/pkg/enums"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContentReviewRepository interface {
	WithTx(tx *query.Query) ContentReviewRepository
	Create(reviewDO *do.ContentReviewDO) error
	// LatestRejectReasonByContentIDs 批量取内容最新一条拒绝理由
	LatestRejectReasonByContentIDs(contentIDs []int64) (map[int64]string, error)
}

type ContentReviewRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	tx  *query.Query
	logx.Logger
}

func NewContentReviewRepository(ctx context.Context, db *orm.DB) ContentReviewRepository {
	return &ContentReviewRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *ContentReviewRepositoryImpl) WithTx(tx *query.Query) ContentReviewRepository {
	return &ContentReviewRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *ContentReviewRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *ContentReviewRepositoryImpl) Create(reviewDO *do.ContentReviewDO) error {
	q := r.getQuery()
	reviewModel := &model.RanFeedContentReview{
		ID:        reviewDO.ID,
		ContentID: reviewDO.ContentID,
		Decision:  reviewDO.Decision,
		Reason:    reviewDO.Reason,
		CreatedBy: reviewDO.CreatedBy,
		UpdatedBy: reviewDO.UpdatedBy,
	}
	return q.RanFeedContentReview.WithContext(r.ctx).Create(reviewModel)
}

// LatestRejectReasonByContentIDs 取每条内容最新一条拒绝审核的理由
func (r *ContentReviewRepositoryImpl) LatestRejectReasonByContentIDs(contentIDs []int64) (map[int64]string, error) {
	res := make(map[int64]string)
	if len(contentIDs) == 0 {
		return res, nil
	}
	q := r.getQuery()
	rows, err := q.RanFeedContentReview.WithContext(r.ctx).
		Select(q.RanFeedContentReview.ContentID, q.RanFeedContentReview.Reason, q.RanFeedContentReview.CreatedAt).
		Where(q.RanFeedContentReview.ContentID.In(contentIDs...)).
		Where(q.RanFeedContentReview.Decision.Eq(int32(content.ReviewDecision_REVIEW_DECISION_REJECT))).
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
