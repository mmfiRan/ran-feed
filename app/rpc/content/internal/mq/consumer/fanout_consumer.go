package consumer

import (
	"context"

	"ran-feed/app/rpc/content/internal/common/component/followfeed"
	"ran-feed/app/rpc/content/internal/mq/event"
	"ran-feed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// FanOutConsumer 消费扇出分批消息 逐批把粉丝写进各自收件箱 独立 group 与在线读隔离
type FanOutConsumer struct {
	followFeed *followfeed.Feed
}

func NewFanOutConsumer(svcCtx *svc.ServiceContext) *FanOutConsumer {
	return &FanOutConsumer{followFeed: svcCtx.FollowFeed}
}

// Consume 整批写收件箱 任一步失败整批返回错误交 kafka 重投 写收件箱幂等可重放
func (c *FanOutConsumer) Consume(ctx context.Context, key, val string) error {
	batch, err := event.UnmarshalFanOutBatch(val)
	if err != nil {
		// 报文损坏重投也无法成功 记日志跳过 不阻塞分区
		logx.WithContext(ctx).Errorf("解析扇出消息失败 跳过 err=%v val=%s", err, val)
		return nil
	}
	if batch == nil || batch.ContentID <= 0 || batch.AuthorID <= 0 || len(batch.FollowerIDs) == 0 {
		return nil
	}
	return c.followFeed.AddFanOut(ctx, batch.FollowerIDs, batch.ContentID, batch.PublishedAt)
}
