package repositories

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/pkg/event/contentevent"
	"ran-feed/pkg/orm"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

// ContentOutboxRepository content 域事件发件箱
type ContentOutboxRepository interface {
	WithTx(tx *query.Query) ContentOutboxRepository
	CreateEvent(evt *contentevent.ContentEvent) error
	// ListUnconsumedEvents 取时间窗内该消费者尚无幂等记录的事件 按 id 升序 afterID 游标分页 供对账补跑
	ListUnconsumedEvents(ctx context.Context, consumer string, from, to time.Time, afterID int64, limit int) ([]*model.RanFeedContentOutbox, error)
	// DeleteEventsBefore 清理保留期外事件
	DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error)
}

type ContentOutboxRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	tx  *query.Query
	logx.Logger
}

func NewContentOutboxRepository(ctx context.Context, db *orm.DB) ContentOutboxRepository {
	return &ContentOutboxRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *ContentOutboxRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *ContentOutboxRepositoryImpl) WithTx(tx *query.Query) ContentOutboxRepository {
	return &ContentOutboxRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

// CreateEvent 创建事件
func (r *ContentOutboxRepositoryImpl) CreateEvent(evt *contentevent.ContentEvent) error {
	payload, err := evt.Marshal()
	if err != nil {
		return err
	}
	q := r.getQuery()
	return q.RanFeedContentOutbox.WithContext(r.ctx).Create(&model.RanFeedContentOutbox{
		EventID:     strconv.FormatInt(snowflake.GenID(), 10),
		EventType:   int32(evt.EventType),
		AggregateID: evt.ContentID,
		Payload:     payload,
	})
}

// ListUnconsumedEvents 左连去重表取该消费者未记录的事件 时间窗内按 id 升序游标推进
func (r *ContentOutboxRepositoryImpl) ListUnconsumedEvents(ctx context.Context, consumer string, from, to time.Time, afterID int64, limit int) ([]*model.RanFeedContentOutbox, error) {
	if consumer == "" || limit <= 0 {
		return nil, nil
	}
	rows := make([]*model.RanFeedContentOutbox, 0, limit)
	err := r.db.DB.WithContext(ctx).
		Table(model.TableNameRanFeedContentOutbox+" AS o").
		Select("o.*").
		Joins("LEFT JOIN ran_feed_mq_consume_dedup AS d ON d.consumer = ? AND d.event_id = o.event_id", consumer).
		Where("d.id IS NULL").
		Where("o.created_at >= ? AND o.created_at < ?", from, to).
		Where("o.id > ?", afterID).
		Order("o.id ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// DeleteEventsBefore 清理保留期外事件 保留期须长于去重表保留期
func (r *ContentOutboxRepositoryImpl) DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error) {
	info := r.db.DB.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&model.RanFeedContentOutbox{})
	return info.RowsAffected, info.Error
}
