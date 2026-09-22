package repositories

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/pkg/event"
	"ran-feed/pkg/orm"
	"ran-feed/pkg/snowflake"

	"github.com/zeromicro/go-zero/core/logx"
)

// ContentOutboxRepository content 域事件发件箱
type ContentOutboxRepository interface {
	WithTx(tx *query.Query) ContentOutboxRepository
	CreateEvent(evt *event.ContentEvent) error
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
func (r *ContentOutboxRepositoryImpl) CreateEvent(evt *event.ContentEvent) error {
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
