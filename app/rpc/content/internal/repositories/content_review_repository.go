package repositories

import (
	"context"

	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type ContentReviewRepository interface {
	WithTx(tx *query.Query) ContentReviewRepository
	Create(reviewDO *do.ContentReviewDO) error
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
