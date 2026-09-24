package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"gorm.io/gorm"

	"ran-feed/app/rpc/notification/internal/entity/model"
	"ran-feed/app/rpc/notification/internal/entity/query"
	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/notification/internal/repositories"
	"ran-feed/app/rpc/notification/internal/svc"
	"ran-feed/pkg/event/canal"
	"ran-feed/pkg/event/pipeline"

	// 空导入触发各表策略 init 注册
	_ "ran-feed/app/rpc/notification/internal/mq/consumer/strategy/presence"
)

const (
	consumerName      = "notification.canal_consumer"
	dispatchTimeout   = 5 * time.Second
	notifyPushChannel = "notify:push"
)

// notifyPushPayload dispatch → SSE 层的信号消息 与 front sse.PubSubMessage 对齐
type notifyPushPayload struct {
	RecipientID int64 `json:"recipient_id"`
	Unread      int64 `json:"unread"`
}

// CanalNotificationConsumer 五步管道 解析 → strategy 路由 → 每行 rowEventID dedup 与落库同事务 → 事务外 dispatch(桩)
type CanalNotificationConsumer struct {
	ctx        context.Context
	svcContext *svc.ServiceContext
	logx.Logger
	notifyRepo repositories.NotificationRepository
	strategies *strategy.Registry
}

func NewCanalNotificationConsumer(ctx context.Context, svcContext *svc.ServiceContext) *CanalNotificationConsumer {
	return &CanalNotificationConsumer{
		ctx:        ctx,
		svcContext: svcContext,
		Logger:     logx.WithContext(ctx),
		notifyRepo: repositories.NewNotificationRepository(ctx, svcContext.MysqlDb),
		strategies: strategy.NewDefaultRegistry(),
	}
}

// Consume 收到一条 canal 消息 解析后按表路由 dedup+落库同事务 事务成功后 dispatch 事件外副作用
func (c *CanalNotificationConsumer) Consume(ctx context.Context, key, val string) error {
	logc.Infof(ctx, "收到canal消息(notification): key=%s", key)

	msg, err := canal.Parse(val)
	if err != nil {
		logc.Errorf(ctx, "解析canal消息失败: %v val=%s", err, val)
		return err
	}

	tableStrategy, ok := c.strategies.Get(msg.Table())
	if !ok {
		logc.Infof(ctx, "跳过未监听表: table=%s", msg.RawTable)
		return nil
	}

	// 落库结果收集 recipient 集合供事务外 dispatch 计未读
	recipients := make(map[int64]struct{})
	err = pipeline.RunInTx(ctx, c.svcContext.MysqlDb.DB, consumerName, msg, val,
		func(ctx context.Context, tx *gorm.DB, meta pipeline.RowMeta, row, oldRow map[string]any) error {
			affected, err := c.processRow(ctx, query.Use(tx), meta, tableStrategy, row, oldRow)
			if err != nil {
				return err
			}
			for _, r := range affected {
				recipients[r] = struct{}{}
			}
			return nil
		})
	if err != nil {
		return err
	}

	c.dispatch(recipients)
	return nil
}

// processRow 单行落库 去重由管道负责 落库失败连带事务回滚
// 返回受影响的 recipient 列表供事务外 dispatch(空表示本行未触发通知)
func (c *CanalNotificationConsumer) processRow(ctx context.Context, tx *query.Query, meta pipeline.RowMeta, tableStrategy strategy.TableStrategy, row, oldRow map[string]any) ([]int64, error) {
	events := tableStrategy.ExtractEvents(ctx, meta.Op, row, oldRow)
	if len(events) == 0 {
		return nil, nil
	}

	notifyRepo := c.notifyRepo.WithTx(tx)
	recipients := make([]int64, 0, len(events))
	for _, e := range events {
		if err := c.persistEvent(notifyRepo, e, meta.UpdatedAt); err != nil {
			return nil, err
		}
		recipients = append(recipients, e.RecipientID)
	}
	return recipients, nil
}

// persistEvent 按 event.Action 分派 UpsertAggregate 或 Insert
func (c *CanalNotificationConsumer) persistEvent(notifyRepo repositories.NotificationRepository, e strategy.NotifyEvent, updatedAt time.Time) error {
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	row := &model.RanFeedNotification{
		RecipientID: e.RecipientID,
		ActorID:     e.ActorID,
		NotifyType:  e.NotifyType.Int32(),
		AggKey:      e.AggKey,
		AggCount:    1,
		ContentID:   e.ContentID,
		CommentID:   e.CommentID,
		Snippet:     e.Snippet,
		IsRead:      0,
		CreatedBy:   e.ActorID,
		UpdatedBy:   e.ActorID,
		CreatedAt:   updatedAt,
		UpdatedAt:   updatedAt,
	}
	switch e.Action {
	case strategy.PersistAggregate:
		return notifyRepo.UpsertAggregate(row)
	case strategy.PersistInsertOne:
		return notifyRepo.Insert(row)
	default:
		return nil
	}
}

// dispatch 事务外副作用 事务提交后异步计每个 recipient 未读数并 PUBLISH notify:push
// front SSE 层订阅该 channel 收信号后回拉;PUBLISH 失败非致命 log 由拉取兜底
func (c *CanalNotificationConsumer) dispatch(recipients map[int64]struct{}) {
	if len(recipients) == 0 {
		return
	}
	// 事务外脱离请求 ctx 独立超时 GoSafe 内层 recover
	threading.GoSafe(func() {
		bg, cancel := context.WithTimeout(context.Background(), dispatchTimeout)
		defer cancel()
		repo := repositories.NewNotificationRepository(bg, c.svcContext.MysqlDb)
		for recipient := range recipients {
			unread, err := repo.CountUnread(recipient)
			if err != nil {
				logc.Errorf(bg, "dispatch CountUnread 失败 recipient=%d err=%v", recipient, err)
				continue
			}
			payload, mErr := json.Marshal(notifyPushPayload{RecipientID: recipient, Unread: unread})
			if mErr != nil {
				logc.Errorf(bg, "dispatch payload 序列化失败 recipient=%d err=%v", recipient, mErr)
				continue
			}
			if _, pErr := c.svcContext.Redis.PublishCtx(bg, notifyPushChannel, payload); pErr != nil {
				logc.Errorf(bg, "dispatch PUBLISH 失败 channel=%s recipient=%d err=%v", notifyPushChannel, recipient, pErr)
			}
		}
	})
}
