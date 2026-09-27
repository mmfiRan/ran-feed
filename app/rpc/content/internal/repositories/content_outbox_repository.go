package repositories

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/pkg/event/contentevent"
)

type ContentOutboxRepository interface {
	WithTx(tx *query.Query) ContentOutboxRepository
	CreateEvent(ctx context.Context, evt *contentevent.ContentEvent) error
	// ListUnconsumedEvents 取时间窗内该消费者尚无幂等记录的事件 按 id 升序 afterID 游标分页 供对账补跑
	ListUnconsumedEvents(ctx context.Context, consumer string, from, to time.Time, afterID int64, limit int) ([]*model.RanFeedContentOutbox, error)
	// DeleteEventsBefore 清理保留期外事件
	DeleteEventsBefore(ctx context.Context, before time.Time) (int64, error)
}
