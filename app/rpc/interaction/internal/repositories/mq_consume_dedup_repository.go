package repositories

import (
	"context"
	"errors"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"

	"ran-feed/app/rpc/interaction/internal/entity/model"
	"ran-feed/app/rpc/interaction/internal/entity/query"
	"ran-feed/pkg/orm"
)

type MqConsumeDedupRepository interface {
	WithTx(tx *query.Query) MqConsumeDedupRepository
	// InsertIfAbsent returns (inserted=true,nil) if inserted, (false,nil) if duplicate, (false,err) for other errors.
	InsertIfAbsent(consumer, eventID string) (bool, error)
}

type mqConsumeDedupRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewMqConsumeDedupRepository(ctx context.Context, db *orm.DB) MqConsumeDedupRepository {
	return &mqConsumeDedupRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *mqConsumeDedupRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *mqConsumeDedupRepositoryImpl) WithTx(tx *query.Query) MqConsumeDedupRepository {
	return &mqConsumeDedupRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *mqConsumeDedupRepositoryImpl) InsertIfAbsent(consumer, eventID string) (bool, error) {
	q := r.getQuery()
	if consumer == "" || eventID == "" {
		return false, nil
	}

	record := &model.RanFeedMqConsumeDedup{
		Consumer: consumer,
		EventID:  eventID,
	}

	err := q.RanFeedMqConsumeDedup.WithContext(r.ctx).Create(record)
	if err == nil {
		return true, nil
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return false, nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return false, nil
	}

	return false, err
}
