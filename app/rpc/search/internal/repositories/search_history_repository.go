package repositories

import (
	"context"
	"time"

	"ran-feed/app/rpc/search/internal/entity/model"
	"ran-feed/app/rpc/search/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchHistoryRepository interface {
	WithTx(tx *query.Query) SearchHistoryRepository
	Upsert(userID int64, keyword string) error
	ListRecent(userID int64, limit int) ([]*model.RanFeedSearchHistory, error)
	DeleteOne(userID int64, keyword string) error
	Clear(userID int64) error
}

type searchHistoryRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewSearchHistoryRepository(ctx context.Context, db *orm.DB) SearchHistoryRepository {
	return &searchHistoryRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *searchHistoryRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *searchHistoryRepositoryImpl) WithTx(tx *query.Query) SearchHistoryRepository {
	return &searchHistoryRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

// Upsert 记录一次搜索 同人同词不新增行只刷新时间并复活已清记录 显式置 updated_at 确保提到最前
func (r *searchHistoryRepositoryImpl) Upsert(userID int64, keyword string) error {
	if userID <= 0 || keyword == "" {
		return nil
	}

	now := time.Now()
	q := r.getQuery()
	return q.RanFeedSearchHistory.WithContext(r.ctx).UnderlyingDB().Exec(
		`INSERT INTO ran_feed_search_history
			(user_id, keyword, status, version, is_deleted, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, 10, 1, 0, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			updated_at = VALUES(updated_at),
			is_deleted = 0,
			updated_by = VALUES(updated_by)`,
		userID, keyword, userID, userID, now, now,
	).Error
}

// ListRecent 取最近 limit 条 按更新时间倒序 读时截断无需淘汰
func (r *searchHistoryRepositoryImpl) ListRecent(userID int64, limit int) ([]*model.RanFeedSearchHistory, error) {
	if userID <= 0 || limit <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedSearchHistory.WithContext(r.ctx).
		Where(q.RanFeedSearchHistory.UserID.Eq(userID)).
		Where(q.RanFeedSearchHistory.IsDeleted.Eq(0)).
		Order(q.RanFeedSearchHistory.UpdatedAt.Desc()).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// DeleteOne 软删单条
func (r *searchHistoryRepositoryImpl) DeleteOne(userID int64, keyword string) error {
	if userID <= 0 || keyword == "" {
		return nil
	}

	q := r.getQuery()
	_, err := q.RanFeedSearchHistory.WithContext(r.ctx).
		Where(q.RanFeedSearchHistory.UserID.Eq(userID)).
		Where(q.RanFeedSearchHistory.Keyword.Eq(keyword)).
		Where(q.RanFeedSearchHistory.IsDeleted.Eq(0)).
		Update(q.RanFeedSearchHistory.IsDeleted, 1)
	return err
}

// Clear 软删该用户全部历史
func (r *searchHistoryRepositoryImpl) Clear(userID int64) error {
	if userID <= 0 {
		return nil
	}

	q := r.getQuery()
	_, err := q.RanFeedSearchHistory.WithContext(r.ctx).
		Where(q.RanFeedSearchHistory.UserID.Eq(userID)).
		Where(q.RanFeedSearchHistory.IsDeleted.Eq(0)).
		Update(q.RanFeedSearchHistory.IsDeleted, 1)
	return err
}
