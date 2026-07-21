package consumer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"

	"ran-feed/app/rpc/notification/internal/entity/model"
	"ran-feed/app/rpc/notification/internal/entity/query"
	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/notification/internal/repositories"
	"ran-feed/app/rpc/notification/internal/svc"

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
	dedupRepo  repositories.MqConsumeDedupRepository
	strategies *strategy.Registry
}

func NewCanalNotificationConsumer(ctx context.Context, svcContext *svc.ServiceContext) *CanalNotificationConsumer {
	return &CanalNotificationConsumer{
		ctx:        ctx,
		svcContext: svcContext,
		Logger:     logx.WithContext(ctx),
		notifyRepo: repositories.NewNotificationRepository(ctx, svcContext.MysqlDb),
		dedupRepo:  repositories.NewMqConsumeDedupRepository(ctx, svcContext.MysqlDb),
		strategies: strategy.NewDefaultRegistry(),
	}
}

// Consume 收到一条 canal 消息 解析后按表路由 dedup+落库同事务 事务成功后 dispatch 事件外副作用
func (c *CanalNotificationConsumer) Consume(ctx context.Context, key, val string) error {
	logc.Infof(ctx, "收到canal消息(notification): key=%s", key)

	var msg canalMessage
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		logc.Errorf(ctx, "解析canal消息失败: %v val=%s", err, val)
		return err
	}

	tableStrategy, ok := c.strategies.Get(msg.table())
	if !ok {
		logc.Infof(ctx, "跳过未监听表: table=%s", msg.Table)
		return nil
	}

	eventID := msg.eventID(val)
	if eventID == "" {
		logc.Errorf(ctx, "canal消息event_id为空: table=%s", msg.Table)
		return nil
	}

	meta := rowMeta{
		table:     msg.table(),
		op:        msg.op(),
		eventID:   eventID,
		updatedAt: msg.updatedAt(),
		strategy:  tableStrategy,
	}

	// 落库结果收集 recipient 集合供事务外 dispatch 计未读
	recipients := make(map[int64]struct{})
	err := query.Q.Transaction(func(tx *query.Query) error {
		for i, row := range msg.Data {
			if row == nil {
				continue
			}
			affected, err := c.processRow(ctx, tx, meta, i, row, msg.oldRow(i))
			if err != nil {
				return err
			}
			for _, r := range affected {
				recipients[r] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	c.dispatch(recipients)
	return nil
}

// rowMeta 消息内逐行共享的元信息
type rowMeta struct {
	table     string
	op        string
	eventID   string
	updatedAt time.Time
	strategy  strategy.TableStrategy
}

// processRow 单行 dedup+落库 dedup 命中则跳过 落库失败连带事务回滚
// 返回受影响的 recipient 列表供事务外 dispatch(空表示本行未触发通知)
func (c *CanalNotificationConsumer) processRow(ctx context.Context, tx *query.Query, meta rowMeta, idx int, row, oldRow map[string]interface{}) ([]int64, error) {
	eid := rowEventID(meta.eventID, meta.table, meta.op, row, idx)
	inserted, err := c.dedupRepo.WithTx(tx).InsertIfAbsent(consumerName, eid)
	if err != nil {
		return nil, err
	}
	if !inserted {
		logc.Infof(ctx, "canal 行已处理跳过: rowEventID=%s table=%s", eid, meta.table)
		return nil, nil
	}

	events := meta.strategy.ExtractEvents(ctx, meta.op, row, oldRow)
	if len(events) == 0 {
		return nil, nil
	}

	notifyRepo := c.notifyRepo.WithTx(tx)
	recipients := make([]int64, 0, len(events))
	for _, e := range events {
		if err := c.persistEvent(notifyRepo, e, meta.updatedAt); err != nil {
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
		NotifyType:  e.NotifyType,
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
