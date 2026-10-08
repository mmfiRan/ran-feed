package consumer

import (
	"context"
	"time"

	"ran-feed/app/rpc/content/internal/common/component/feedprojector"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/event/canal"
	"ran-feed/pkg/event/contentevent"
	"ran-feed/pkg/event/dedup"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	outboxTable = "ran_feed_content_outbox"
	opInsert    = "INSERT"
	// 单条事件处理的有界重试次数与退避基数 耗尽后交 kafka 重投 可靠性最终由对账作业兜底
	handleMaxAttempts   = 3
	handleRetryBaseWait = 100 * time.Millisecond
)

// ContentEventConsumer 消费 content 域 outbox 事件 只做协议解析 幂等与重试 业务投影下沉 feedprojector
type ContentEventConsumer struct {
	dedupGate *dedup.Gate
	projector *feedprojector.Projector
}

func NewContentEventConsumer(svcCtx *svc.ServiceContext) *ContentEventConsumer {
	return &ContentEventConsumer{
		dedupGate: dedup.New(svcCtx.MysqlDb.DB),
		projector: svcCtx.FeedProjector,
	}
}

// Consume 解析 canal 消息 仅处理 outbox 表 INSERT 逐行幂等处理
func (c *ContentEventConsumer) Consume(ctx context.Context, key, val string) error {
	msg, err := canal.Parse(val)
	if err != nil {
		logx.WithContext(ctx).Errorf("解析 canal 消息失败 err=%v val=%s", err, val)
		return err
	}
	if msg.Table() != outboxTable || msg.Op() != opInsert {
		return nil
	}

	eventID := msg.EventID(val)
	for i, row := range msg.Data {
		if row == nil {
			continue
		}
		if err = c.processRow(ctx, eventID, msg.Table(), msg.Op(), row, i); err != nil {
			return err
		}
	}
	return nil
}

// processRow 单行处理 Exists 跳过已处理 处理失败返回错误交 kafka 重投 成功后标记 dedup
func (c *ContentEventConsumer) processRow(ctx context.Context, eventID, table, op string, row map[string]any, idx int) error {
	eid := outboxEventKey(eventID, table, op, row, idx)
	seen, err := c.dedupGate.Exists(ctx, contentconsts.ContentEventConsumerName, eid)
	if err != nil {
		return err
	}
	if seen {
		return nil
	}

	payload := canal.ParseString(row["payload"])
	evt, err := contentevent.UnmarshalContentEvent(payload)
	if err != nil {
		logx.WithContext(ctx).Errorf("解析 content 事件失败 跳过 payload=%s err=%v", payload, err)
		return nil
	}

	if err = c.handleWithRetry(ctx, evt); err != nil {
		return err
	}

	if _, err := c.dedupGate.InsertIfAbsent(ctx, contentconsts.ContentEventConsumerName, eid); err != nil {
		logx.WithContext(ctx).Errorf("插入去重表失败 eid=%s err=%v", eid, err)
	}
	return nil
}

func outboxEventKey(eventID, table, op string, row map[string]any, idx int) string {
	if eid := canal.ParseString(row["event_id"]); eid != "" {
		return eid
	}
	return canal.RowEventID(eventID, table, op, row, idx)
}

// handleWithRetry 有界重试 指数退避 耗尽后返回最后错误交 kafka 重投
func (c *ContentEventConsumer) handleWithRetry(ctx context.Context, evt *contentevent.ContentEvent) error {
	return retryWithBackoff(ctx, handleMaxAttempts, handleRetryBaseWait, func() error {
		return c.projector.Apply(ctx, evt)
	})
}

// retryWithBackoff 指数退避重试
func retryWithBackoff(ctx context.Context, attempts int, base time.Duration, fn func() error) error {
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt == attempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(base << attempt):
		}
	}
	return err
}
