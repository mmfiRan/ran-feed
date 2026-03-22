package repositories

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ran-feed/app/rpc/count/internal/entity/model"
	"ran-feed/app/rpc/count/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type CountValueRepository interface {
	WithTx(tx *query.Query) CountValueRepository
	Get(bizType int32, targetType int32, targetID int64) (*model.RanFeedCountValue, error)
	BatchGet(bizType int32, targetType int32, targetIDs []int64) (map[int64]*model.RanFeedCountValue, error)
	SumByOwner(bizType int32, targetType int32, ownerID int64) (int64, error)
	UpsertValue(bizType int32, targetType int32, targetID int64, value int64, updatedAt time.Time) error
	UpdateDelta(bizType int32, targetType int32, targetID int64, delta int64, updatedAt time.Time) (int64, error)
	UpdateDeltaWithOwner(bizType int32, targetType int32, targetID int64, ownerID int64, delta int64, updatedAt time.Time) (int64, error)
}

type countValueRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewCountValueRepository(ctx context.Context, db *orm.DB) CountValueRepository {
	return &countValueRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *countValueRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *countValueRepositoryImpl) WithTx(tx *query.Query) CountValueRepository {
	return &countValueRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *countValueRepositoryImpl) Get(bizType int32, targetType int32, targetID int64) (*model.RanFeedCountValue, error) {
	if bizType <= 0 || targetType <= 0 || targetID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedCountValue.WithContext(r.ctx).
		Where(q.RanFeedCountValue.BizType.Eq(bizType)).
		Where(q.RanFeedCountValue.TargetType.Eq(targetType)).
		Where(q.RanFeedCountValue.TargetID.Eq(targetID)).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

func (r *countValueRepositoryImpl) BatchGet(bizType int32, targetType int32, targetIDs []int64) (map[int64]*model.RanFeedCountValue, error) {
	res := make(map[int64]*model.RanFeedCountValue, len(targetIDs))
	if bizType <= 0 || targetType <= 0 || len(targetIDs) == 0 {
		return res, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedCountValue.WithContext(r.ctx).
		Where(q.RanFeedCountValue.BizType.Eq(bizType)).
		Where(q.RanFeedCountValue.TargetType.Eq(targetType)).
		Where(q.RanFeedCountValue.TargetID.In(targetIDs...)).
		Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		res[row.TargetID] = row
	}
	return res, nil
}

func (r *countValueRepositoryImpl) SumByOwner(bizType int32, targetType int32, ownerID int64) (int64, error) {
	if bizType <= 0 || targetType <= 0 || ownerID <= 0 {
		return 0, nil
	}

	q := r.getQuery()
	var sum int64
	err := q.RanFeedCountValue.WithContext(r.ctx).UnderlyingDB().
		Model(&model.RanFeedCountValue{}).
		Select("COALESCE(SUM(value),0)").
		Where("biz_type = ? AND target_type = ? AND owner_id = ?", bizType, targetType, ownerID).
		Scan(&sum).Error
	if err != nil {
		return 0, err
	}
	return sum, nil
}

func (r *countValueRepositoryImpl) UpsertValue(bizType int32, targetType int32, targetID int64, value int64, updatedAt time.Time) error {
	if bizType <= 0 || targetType <= 0 || targetID <= 0 {
		return nil
	}

	q := r.getQuery()
	row := &model.RanFeedCountValue{
		BizType:    bizType,
		TargetType: targetType,
		TargetID:   targetID,
		Value:      value,
		Version:    0,
		CreatedAt:  updatedAt,
		UpdatedAt:  updatedAt,
	}

	return q.RanFeedCountValue.WithContext(r.ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "biz_type"}, {Name: "target_type"}, {Name: "target_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"value": value, "updated_at": updatedAt}),
		}).
		Create(row)
}

func (r *countValueRepositoryImpl) UpdateDelta(bizType int32, targetType int32, targetID int64, delta int64, updatedAt time.Time) (int64, error) {
	if bizType <= 0 || targetType <= 0 || targetID <= 0 || delta == 0 {
		return 0, nil
	}

	q := r.getQuery()
	initValue := delta
	db := q.RanFeedCountValue.WithContext(r.ctx).UnderlyingDB()
	err := db.Exec(
		`INSERT INTO ran_feed_count_value
			(biz_type, target_type, target_id, value, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			value = GREATEST(value + VALUES(value), 0),
			version = version + 1,
			updated_at = VALUES(updated_at)`,
		bizType,
		targetType,
		targetID,
		initValue,
		1,
		updatedAt,
		updatedAt,
	).Error
	if err != nil {
		return 0, err
	}
	return 0, nil
}

func (r *countValueRepositoryImpl) UpdateDeltaWithOwner(
	bizType int32,
	targetType int32,
	targetID int64,
	ownerID int64,
	delta int64,
	updatedAt time.Time,
) (int64, error) {
	if bizType <= 0 || targetType <= 0 || targetID <= 0 || delta == 0 {
		return 0, nil
	}

	q := r.getQuery()
	initValue := delta
	db := q.RanFeedCountValue.WithContext(r.ctx).UnderlyingDB()
	err := db.Exec(
		`INSERT INTO ran_feed_count_value
			(biz_type, target_type, target_id, owner_id, value, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			value = GREATEST(value + VALUES(value), 0),
			version = version + 1,
			updated_at = VALUES(updated_at),
			owner_id = CASE WHEN owner_id = 0 THEN VALUES(owner_id) ELSE owner_id END`,
		bizType,
		targetType,
		targetID,
		ownerID,
		initValue,
		1,
		updatedAt,
		updatedAt,
	).Error
	if err != nil {
		return 0, err
	}
	return 0, nil
}
