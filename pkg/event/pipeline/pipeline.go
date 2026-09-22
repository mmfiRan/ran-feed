// Package pipeline canal 逐行去重与业务落库同事务
package pipeline

import (
	"context"
	"time"

	"gorm.io/gorm"

	"ran-feed/pkg/event/canal"
	"ran-feed/pkg/event/dedup"
)

// RowMeta 一条消息内逐行处理共享的元信息
type RowMeta struct {
	Table     string
	Op        string
	EventID   string
	UpdatedAt time.Time
	Index     int
}

// RowHandler 事务内单行业务处理 调用时该行已通过去重
type RowHandler func(ctx context.Context, tx *gorm.DB, meta RowMeta, row, oldRow map[string]any) error

// RunInTx 逐行去重与业务落库同事务 保证幂等
func RunInTx(ctx context.Context, db *gorm.DB, consumerName string, msg *canal.Message, raw string, h RowHandler) error {
	eventID := msg.EventID(raw)
	if eventID == "" {
		return nil
	}

	table, op, updatedAt := msg.Table(), msg.Op(), msg.UpdatedAt()
	return db.Transaction(func(tx *gorm.DB) error {
		gate := dedup.New(tx)
		for i, row := range msg.Data {
			if row == nil {
				continue
			}
			inserted, err := gate.InsertIfAbsent(ctx, consumerName, canal.RowEventID(eventID, table, op, row, i))
			if err != nil {
				return err
			}
			if !inserted {
				continue
			}
			meta := RowMeta{
				Table:     table,
				Op:        op,
				EventID:   eventID,
				UpdatedAt: updatedAt,
				Index:     i,
			}
			if err = h(ctx, tx, meta, row, msg.OldRow(i)); err != nil {
				return err
			}
		}
		return nil
	})
}
