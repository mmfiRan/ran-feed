package consumer

import (
	"context"
	"strconv"
	"time"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/common/utils/contentcache"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	contentenums "ran-feed/pkg/enums/content"
	"ran-feed/pkg/event/canal"
	"ran-feed/pkg/event/contentevent"
	"ran-feed/pkg/event/dedup"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// FeedConsumerName 消费者标识
	FeedConsumerName = "content.feed_consumer"
	outboxTable      = "ran_feed_content_outbox"
	opInsert         = "INSERT"
	// 单条事件处理的有界重试次数与退避基数 耗尽后交 kafka 重投 可靠性最终由对账作业兜底
	handleMaxAttempts   = 3
	handleRetryBaseWait = 100 * time.Millisecond
)

// ContentEventConsumer 消费 content 域 outbox 事件
type ContentEventConsumer struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	dedupGate   *dedup.Gate
	contentRepo repositories.ContentRepository
}

func NewContentEventConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *ContentEventConsumer {
	return &ContentEventConsumer{
		ctx:         ctx,
		svcCtx:      svcCtx,
		Logger:      logx.WithContext(ctx),
		dedupGate:   dedup.New(svcCtx.MysqlDb.DB),
		contentRepo: repositories.NewContentRepository(ctx, svcCtx.MysqlDb),
	}
}

// Consume 解析 canal 消息 仅处理 outbox 表 INSERT 逐行幂等处理
func (c *ContentEventConsumer) Consume(ctx context.Context, key, val string) error {
	msg, err := canal.Parse(val)
	if err != nil {
		c.Errorf("解析 canal 消息失败 err=%v val=%s", err, val)
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
	seen, err := c.dedupGate.Exists(ctx, FeedConsumerName, eid)
	if err != nil {
		return err
	}
	if seen {
		return nil
	}

	payload := canal.ParseString(row["payload"])
	evt, err := contentevent.UnmarshalContentEvent(payload)
	if err != nil {
		c.Errorf("解析 content 事件失败 跳过 payload=%s err=%v", payload, err)
		return nil
	}

	if err = c.handleWithRetry(ctx, evt); err != nil {
		return err
	}

	if _, err := c.dedupGate.InsertIfAbsent(ctx, FeedConsumerName, eid); err != nil {
		c.Errorf("插入去重表失败 eid=%s err=%v", eid, err)
	}
	return nil
}

// Replay 对账只有 outbox 对账定时任务会调
func (c *ContentEventConsumer) Replay(ctx context.Context, eventID string, evt *contentevent.ContentEvent) error {
	if err := c.handleWithRetry(ctx, evt); err != nil {
		return err
	}
	if _, err := c.dedupGate.InsertIfAbsent(ctx, FeedConsumerName, eventID); err != nil {
		c.Errorf("对账补跑落去重表失败 eid=%s err=%v", eventID, err)
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
		return c.handle(ctx, evt)
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

// handle 按事件类型分发处理
func (c *ContentEventConsumer) handle(ctx context.Context, evt *contentevent.ContentEvent) error {
	switch evt.EventType {
	case contentenums.EventTypePublished, contentenums.EventTypeRestored:
		return c.syncPublished(ctx, evt)
	case contentenums.EventTypeDeleted, contentenums.EventTypeTakenDown:
		return c.cleanup(ctx, evt.ContentID, evt.AuthorID)
	default:
		return nil
	}
}

// syncPublished 触发 feed 流的发布,先失效二级缓存,再处理一级缓存
func (c *ContentEventConsumer) syncPublished(ctx context.Context, evt *contentevent.ContentEvent) error {
	row, err := c.contentRepo.GetDetailByID(evt.ContentID)
	if err != nil {
		return err
	}
	if row == nil ||
		row.Status != contentEnum.ContentStatusPublished.Int32() ||
		row.Visibility != contentEnum.VisibilityPublic.Int32() {
		return c.cleanup(ctx, evt.ContentID, evt.AuthorID)
	}
	// 先失效 L2 再写 L1
	if err = contentcache.Invalidate(ctx, c.svcCtx.Redis, evt.ContentID); err != nil {
		return err
	}
	var publishedAtMillis int64
	if row.PublishedAt != nil {
		publishedAtMillis = row.PublishedAt.UnixMilli()
	}
	return c.svcCtx.FeedPublisher.Publish(ctx, evt.ContentID, row.UserID, publishedAtMillis, contentEnum.VisibilityEnum(row.Visibility))
}

// cleanup 删除或下架清理热榜主榜 + 作者发件箱 + 失效 L2 follower inbox
func (c *ContentEventConsumer) cleanup(ctx context.Context, contentID, authorID int64) error {
	contentIDStr := strconv.FormatInt(contentID, 10)
	if _, err := c.svcCtx.Redis.ZremCtx(ctx, rediskey.RedisFeedHotGlobalKey, contentIDStr); err != nil {
		return err
	}
	if authorID > 0 {
		if _, err := c.svcCtx.Redis.ZremCtx(ctx, rediskey.BuildUserPublishFeedKey(authorID), contentIDStr); err != nil {
			return err
		}
	}
	return contentcache.Invalidate(ctx, c.svcCtx.Redis, contentID)
}
