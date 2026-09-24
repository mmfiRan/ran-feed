package repositories

import (
	"context"

	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
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
	// LatestReasonByContentIDs 批量取内容最新一条指定决策的理由
	LatestReasonByContentIDs(contentIDs []int64, decisions []contentEnum.ReviewDecisionEnum) (map[int64]string, error)
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
		Decision:  reviewDO.Decision.Int32(),
		Reason:    reviewDO.Reason,
		CreatedBy: reviewDO.CreatedBy,
		UpdatedBy: reviewDO.UpdatedBy,
	}
	return q.RanFeedContentReview.WithContext(r.ctx).Create(reviewModel)
}

// LatestReasonByContentIDs 取每条内容最新一条指定决策(拒绝/下架等)的理由
func (r *ContentReviewRepositoryImpl) LatestReasonByContentIDs(contentIDs []int64, decisions []contentEnum.ReviewDecisionEnum) (map[int64]string, error) {
	res := make(map[int64]string)
	if len(contentIDs) == 0 || len(decisions) == 0 {
		return res, nil
	}
	q := r.getQuery()
	codes := make([]int32, 0, len(decisions))
	for _, d := range decisions {
		codes = append(codes, d.Int32())
	}
	rows, err := q.RanFeedContentReview.WithContext(r.ctx).
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
