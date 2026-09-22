package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/notification/internal/entity/model"
	"ran-feed/app/rpc/notification/internal/entity/query"
	"ran-feed/app/rpc/notification/internal/repositories"
	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/app/rpc/notification/notification"
	contentenums "ran-feed/pkg/enums/content"
	"ran-feed/pkg/event"
)

const (
	contentReviewConsumerName = "notification.content_review_consumer"
	contentOutboxTable        = "ran_feed_content_outbox"
	snippetMaxRunes           = 140
)

// ContentReviewConsumer 消费 content 域 outbox 事件 把审核结果通知作者
// 与主 canal 消费者独立 topic 独立 group dedup 与落库同事务
type ContentReviewConsumer struct {
	ctx        context.Context
	svcContext *svc.ServiceContext
	logx.Logger
	notifyRepo repositories.NotificationRepository
	dedupRepo  repositories.MqConsumeDedupRepository
}

func NewContentReviewConsumer(ctx context.Context, svcContext *svc.ServiceContext) *ContentReviewConsumer {
	return &ContentReviewConsumer{
		ctx:        ctx,
		svcContext: svcContext,
		Logger:     logx.WithContext(ctx),
		notifyRepo: repositories.NewNotificationRepository(ctx, svcContext.MysqlDb),
		dedupRepo:  repositories.NewMqConsumeDedupRepository(ctx, svcContext.MysqlDb),
	}
}

// Consume 解析 outbox canal 消息 逐行 dedup 与落库同事务 事务成功后 dispatch
func (c *ContentReviewConsumer) Consume(ctx context.Context, key, val string) error {
	var msg canalMessage
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		logc.Errorf(ctx, "解析 canal 消息失败(content-review): %v val=%s", err, val)
		return err
	}
	if msg.table() != contentOutboxTable || msg.op() != "INSERT" {
		return nil
	}

	eventID := msg.eventID(val)
	updatedAt := msg.updatedAt()
	recipients := make(map[int64]struct{})

	err := query.Q.Transaction(func(tx *query.Query) error {
		for i, row := range msg.Data {
			if row == nil {
				continue
			}
			eid := rowEventID(eventID, msg.table(), msg.op(), row, i)
			inserted, err := c.dedupRepo.WithTx(tx).InsertIfAbsent(contentReviewConsumerName, eid)
			if err != nil {
				return err
			}
			if !inserted {
				continue
			}
			evt, err := event.UnmarshalContentEvent(payloadOf(row))
			if err != nil {
				logc.Errorf(ctx, "解析 content 事件失败 跳过 err=%v", err)
				continue
			}
			notifyRow, ok := buildReviewNotification(evt, updatedAt)
			if !ok {
				continue
			}
			if err := c.notifyRepo.WithTx(tx).UpsertReview(notifyRow); err != nil {
				return err
			}
			recipients[evt.AuthorID] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return err
	}

	c.dispatch(recipients)
	return nil
}

// dispatch 事务外推送未读数 供 SSE 实时更新 失败只记日志
func (c *ContentReviewConsumer) dispatch(recipients map[int64]struct{}) {
	for recipientID := range recipients {
		c.pushUnread(recipientID)
	}
}

func (c *ContentReviewConsumer) pushUnread(recipientID int64) {
	unread, err := c.notifyRepo.CountUnread(recipientID)
	if err != nil {
		c.Errorf("审核通知查未读数失败 recipientID=%d err=%v", recipientID, err)
		return
	}
	payload, err := json.Marshal(notifyPushPayload{RecipientID: recipientID, Unread: unread})
	if err != nil {
		return
	}
	if _, err := c.svcContext.Redis.PublishCtx(c.ctx, notifyPushChannel, string(payload)); err != nil {
		c.Errorf("审核通知推送失败 recipientID=%d err=%v", recipientID, err)
	}
}

// buildReviewNotification 事件转通知行 非审核类事件返回 false
func buildReviewNotification(evt *event.ContentEvent, at time.Time) (*model.RanFeedNotification, bool) {
	var snippet string
	switch evt.EventType {
	case contentenums.EventTypePublished:
		snippet = "内容已通过审核并发布"
	case contentenums.EventTypeRestored:
		snippet = "内容已恢复上架"
	case contentenums.EventTypeRejected:
		snippet = "内容未通过审核" + reasonSuffix(evt.Reason)
	case contentenums.EventTypeTakenDown:
		snippet = "内容已被下架" + reasonSuffix(evt.Reason)
	default:
		return nil, false
	}
	return &model.RanFeedNotification{
		RecipientID: evt.AuthorID,
		ActorID:     0,
		NotifyType:  int32(notification.NotifyType_NOTIFY_TYPE_CONTENT_REVIEW),
		AggKey:      fmt.Sprintf("RV:%d", evt.ContentID),
		AggCount:    1,
		ContentID:   evt.ContentID,
		Snippet:     truncateRunes(snippet, snippetMaxRunes),
		CreatedAt:   at,
		UpdatedAt:   at,
	}, true
}

// reasonSuffix 有理由则前缀冒号无理由返回空串 避免尾随空格
func reasonSuffix(reason string) string {
	if reason == "" {
		return ""
	}
	return " " + reason
}

// payloadOf 取 outbox 行 payload 字段 canal flatMessage 列值为字符串
func payloadOf(row map[string]interface{}) string {
	if v, ok := row["payload"].(string); ok {
		return v
	}
	return ""
}

// truncateRunes 按 rune 截断防止多字节被截坏
func truncateRunes(s string, max int) string {
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	return string(rs[:max])
}
