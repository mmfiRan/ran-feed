package consumer

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/contentcache"
	"ran-feed/app/rpc/content/internal/svc"
	contentenums "ran-feed/pkg/enums/content"
	"ran-feed/pkg/event"
	"ran-feed/pkg/event/canal"
	"ran-feed/pkg/event/dedup"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	feedConsumerName = "content.feed_consumer"
	outboxTable      = "ran_feed_content_outbox"
	opInsert         = "INSERT"
)

// ContentEventConsumer 消费 content 域 outbox 事件
type ContentEventConsumer struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	dedupGate *dedup.Gate
}

func NewContentEventConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *ContentEventConsumer {
	return &ContentEventConsumer{
		ctx:       ctx,
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		dedupGate: dedup.New(svcCtx.MysqlDb.DB),
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
	eid := canal.RowEventID(eventID, table, op, row, idx)
	seen, err := c.dedupGate.Exists(ctx, feedConsumerName, eid)
	if err != nil {
		return err
	}
	if seen {
		return nil
	}

	payload := canal.ParseString(row["payload"])
	evt, err := event.UnmarshalContentEvent(payload)
	if err != nil {
		c.Errorf("解析 content 事件失败 跳过 payload=%s err=%v", payload, err)
		return nil
	}

	if err = c.handle(ctx, evt); err != nil {
		return err
	}

	if _, err := c.dedupGate.InsertIfAbsent(ctx, feedConsumerName, eid); err != nil {
		c.Errorf("插入去重表失败 eid=%s err=%v", eid, err)
	}
	return nil
}

// handle 按事件类型分发处理
func (c *ContentEventConsumer) handle(ctx context.Context, evt *event.ContentEvent) error {
	switch evt.EventType {
	case contentenums.EventTypePublished, contentenums.EventTypeRestored:
		if err := c.svcCtx.FeedPublisher.Publish(ctx, evt.ContentID, evt.AuthorID, evt.PublishedAt, content.Visibility(evt.Visibility)); err != nil {
			return err
		}
		return contentcache.Invalidate(ctx, c.svcCtx.Redis, evt.ContentID)
	case contentenums.EventTypeDeleted, contentenums.EventTypeTakenDown:
		return c.cleanup(ctx, evt.ContentID, evt.AuthorID)
	default:
		return nil
	}
}

// cleanup 删除或下架清理 ZREM 热榜主榜 + 作者发件箱 + 精确失效 L2 follower inbox 与快照靠 L2 读时重构缓存
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
