package contentoutboxrepo

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/pkg/event/contentevent"
	"ran-feed/pkg/snowflake"
)

var _ repositories.ContentOutboxRepository = (*Repository)(nil)

type Repository struct {
	tx *query.Query
}

func New() *Repository {
	return &Repository{}
}

func (r *Repository) WithTx(tx *query.Query) repositories.ContentOutboxRepository {
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

func (r *Repository) CreateEvent(ctx context.Context, evt *contentevent.ContentEvent) error {
	payload, err := evt.Marshal()
	if err != nil {
		return err
	}
	q := r.getQuery()
	return q.RanFeedContentOutbox.WithContext(ctx).Create(&model.RanFeedContentOutbox{
		EventID:     strconv.FormatInt(snowflake.GenID(), 10),
		EventType:   int32(evt.EventType),
		AggregateID: evt.ContentID,
		Payload:     payload,
	})
}

func (r *Repository) ListUnconsumedEvents(ctx context.Context, consumer string, from, to time.Time, afterID int64, limit int) ([]*model.RanFeedContentOutbox, error) {
	if consumer == "" || limit <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	outbox := q.RanFeedContentOutbox
	dedup := q.RanFeedMqConsumeDedup
	return outbox.WithContext(ctx).
		Select(outbox.ALL).
		LeftJoin(dedup, outbox.EventID.EqCol(dedup.EventID), dedup.Consumer.Eq(consumer)).
		Where(dedup.ID.IsNull()).
		Where(outbox.CreatedAt.Gte(from), outbox.CreatedAt.Lt(to)).
		Where(outbox.ID.Gt(afterID)).
		Order(outbox.ID).
		Limit(limit).
		Find()
}

func (r *Repository) DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error) {
	q := r.getQuery().RanFeedContentOutbox
	info, err := q.WithContext(ctx).Where(q.CreatedAt.Lt(before)).Delete()
	return info.RowsAffected, err
}
