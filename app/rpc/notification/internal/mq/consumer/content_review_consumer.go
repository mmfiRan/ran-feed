package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"gorm.io/gorm"

	notifyenum "ran-feed/app/rpc/notification/internal/common/enums"
	"ran-feed/app/rpc/notification/internal/entity/model"
	"ran-feed/app/rpc/notification/internal/entity/query"
	"ran-feed/app/rpc/notification/internal/repositories"
	"ran-feed/app/rpc/notification/internal/svc"
	contentenums "ran-feed/pkg/enums/content"
	"ran-feed/pkg/event/canal"
	"ran-feed/pkg/event/contentevent"
	"ran-feed/pkg/event/pipeline"
)

const (
	contentReviewConsumerName = "notification.content_review_consumer"
	contentOutboxTable        = "ran_feed_content_outbox"
	opInsert                  = "INSERT"
	snippetMaxRunes           = 140
)

// ContentReviewConsumer 消费 content 域 outbox 事件 把审核结果通知作者
// 与主 canal 消费者独立 topic 独立 group dedup 与落库同事务
type ContentReviewConsumer struct {
	ctx        context.Context
	svcContext *svc.ServiceContext
	logx.Logger
	notifyRepo repositories.NotificationRepository
}

func NewContentReviewConsumer(ctx context.Context, svcContext *svc.ServiceContext) *ContentReviewConsumer {
	return &ContentReviewConsumer{
		ctx:        ctx,
		svcContext: svcContext,
		Logger:     logx.WithContext(ctx),
		notifyRepo: repositories.NewNotificationRepository(ctx, svcContext.MysqlDb),
	}
}

// Consume 解析 outbox canal 消息 逐行 dedup 与落库同事务 事务成功后 dispatch
func (c *ContentReviewConsumer) Consume(ctx context.Context, key, val string) error {
	msg, err := canal.Parse(val)
	if err != nil {
		logc.Errorf(ctx, "解析 canal 消息失败(content-review): %v val=%s", err, val)
		return err
	}
	if msg.Table() != contentOutboxTable || msg.Op() != opInsert {
		return nil
	}

	recipients := make(map[int64]struct{})
	err = pipeline.RunInTx(ctx, c.svcContext.MysqlDb.DB, contentReviewConsumerName, msg, val,
		func(ctx context.Context, tx *gorm.DB, meta pipeline.RowMeta, row, oldRow map[string]any) error {
			evt, err := contentevent.UnmarshalContentEvent(canal.ParseString(row["payload"]))
			if err != nil {
				logc.Errorf(ctx, "解析 content 事件失败 跳过 err=%v", err)
				return nil
			}
			notifyRow, ok := buildReviewNotification(evt, meta.UpdatedAt)
			if !ok {
				return nil
			}
			if err := c.notifyRepo.WithTx(query.Use(tx)).UpsertReview(notifyRow); err != nil {
				return err
			}
			recipients[evt.AuthorID] = struct{}{}
			return nil
		})
	if err != nil {
		return err
	}

	c.dispatch(recipients)
	return nil
}

// dispatch 事务外推送未读数 供 SSE 实时更新 脱离请求 ctx 独立超时 失败只记日志
func (c *ContentReviewConsumer) dispatch(recipients map[int64]struct{}) {
	if len(recipients) == 0 {
		return
	}
	threading.GoSafe(func() {
		bg, cancel := context.WithTimeout(context.Background(), dispatchTimeout)
		defer cancel()
		repo := repositories.NewNotificationRepository(bg, c.svcContext.MysqlDb)
		for recipientID := range recipients {
			c.pushUnread(bg, repo, recipientID)
		}
	})
}

func (c *ContentReviewConsumer) pushUnread(ctx context.Context, repo repositories.NotificationRepository, recipientID int64) {
	unread, err := repo.CountUnread(recipientID)
	if err != nil {
		c.Errorf("审核通知查未读数失败 recipientID=%d err=%v", recipientID, err)
		return
	}
	payload, err := json.Marshal(notifyPushPayload{RecipientID: recipientID, Unread: unread})
	if err != nil {
		return
	}
	if _, err := c.svcContext.Redis.PublishCtx(ctx, notifyPushChannel, string(payload)); err != nil {
		c.Errorf("审核通知推送失败 recipientID=%d err=%v", recipientID, err)
	}
}

// buildReviewNotification 事件转通知行 非审核类事件返回 false
func buildReviewNotification(evt *contentevent.ContentEvent, at time.Time) (*model.RanFeedNotification, bool) {
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
		NotifyType:  notifyenum.NotifyTypeContentReview.Int32(),
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

// truncateRunes 按 rune 截断防止多字节被截坏
func truncateRunes(s string, max int) string {
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	return string(rs[:max])
}
