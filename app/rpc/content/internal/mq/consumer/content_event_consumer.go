package consumer

import (
	"context"
	"encoding/json"
	"strconv"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/contentcache"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	contentenums "ran-feed/pkg/enums/content"
	"ran-feed/pkg/event"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	feedConsumerName = "content.feed_consumer"
	outboxTable      = "ran_feed_content_outbox"
)

// ContentEventConsumer 消费 content 域 outbox 事件 做 feed 扇出与清理
// 副作用是 Redis 无法与 dedup 落库共事务 故处理幂等 先 Exists 跳过 成功后再标记
type ContentEventConsumer struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	dedupRepo repositories.MqConsumeDedupRepository
}

func NewContentEventConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *ContentEventConsumer {
	return &ContentEventConsumer{
		ctx:       ctx,
		svcCtx:    svcCtx,
		Logger:    logx.WithContext(ctx),
		dedupRepo: repositories.NewMqConsumeDedupRepository(ctx, svcCtx.MysqlDb),
	}
}

// Consume 解析 canal 消息 仅处理 outbox 表 INSERT 逐行幂等处理
func (c *ContentEventConsumer) Consume(ctx context.Context, key, val string) error {
	var msg canalMessage
	if err := json.Unmarshal([]byte(val), &msg); err != nil {
		c.Errorf("解析 canal 消息失败 err=%v val=%s", err, val)
		return err
	}
	if msg.table() != outboxTable || msg.op() != "INSERT" {
		return nil
	}

	eventID := msg.eventID(val)
	for i, row := range msg.Data {
		if row == nil {
			continue
		}
		if err := c.processRow(ctx, eventID, msg.table(), msg.op(), row, i); err != nil {
			return err
		}
	}
	return nil
}

// processRow 单行处理 Exists 跳过已处理 处理失败返回错误交 kafka 重投 成功后标记 dedup
func (c *ContentEventConsumer) processRow(ctx context.Context, eventID, table, op string, row map[string]interface{}, idx int) error {
	eid := rowEventID(eventID, table, op, row, idx)
	seen, err := c.dedupRepo.Exists(feedConsumerName, eid)
	if err != nil {
		return err
	}
	if seen {
		return nil
	}

	evt, err := event.UnmarshalContentEvent(stringField(row["payload"]))
	if err != nil {
		c.Errorf("解析 content 事件失败 跳过 payload=%s err=%v", stringField(row["payload"]), err)
		return nil
	}

	if err := c.handle(ctx, evt); err != nil {
		return err
	}

	if _, err := c.dedupRepo.InsertIfAbsent(feedConsumerName, eid); err != nil {
		c.Errorf("标记 dedup 失败 幂等可重放 eid=%s err=%v", eid, err)
	}
	return nil
}

// handle 按事件类型派发副作用 恢复上架当发布处理重进 feed
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

// cleanup 删除或下架清理 ZREM 热榜主榜 + 作者发件箱 + 精确失效 L2 follower inbox 与快照靠 L2 读时自愈
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
