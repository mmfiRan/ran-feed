package repositories

import (
	"context"
	"errors"
	"strings"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

// MqConsumeDedupRepository 消费者行级去重闸 复用公共 ran_feed_mq_consume_dedup
type MqConsumeDedupRepository interface {
	WithTx(tx *query.Query) MqConsumeDedupRepository
	InsertIfAbsent(consumer, eventID string) (bool, error)
	// Exists 是否已处理 用于幂等副作用 处理成功后再 InsertIfAbsent 标记
	Exists(consumer, eventID string) (bool, error)
}

type mqConsumeDedupRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	tx  *query.Query
	logx.Logger
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

// InsertIfAbsent 插入成功返回 true 首次处理 命中唯一键返回 false 已处理过
func (r *mqConsumeDedupRepositoryImpl) InsertIfAbsent(consumer, eventID string) (bool, error) {
	if consumer == "" || eventID == "" {
		return false, nil
	}

	q := r.getQuery()
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

// Exists 查是否已处理过
func (r *mqConsumeDedupRepositoryImpl) Exists(consumer, eventID string) (bool, error) {
	if consumer == "" || eventID == "" {
		return false, nil
	}
	q := r.getQuery()
	cnt, err := q.RanFeedMqConsumeDedup.WithContext(r.ctx).
		Where(q.RanFeedMqConsumeDedup.Consumer.Eq(consumer)).
		Where(q.RanFeedMqConsumeDedup.EventID.Eq(eventID)).
		Count()
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}
