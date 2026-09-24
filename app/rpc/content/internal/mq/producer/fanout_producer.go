package producer

import (
	"context"
	"fmt"

	"ran-feed/app/rpc/content/internal/mq/event"

	"github.com/zeromicro/go-queue/kq"
)

// FanOutProducer 扇出分批消息生产者
type FanOutProducer struct {
	pusher *kq.Pusher
}

func NewFanOutProducer(pusher *kq.Pusher) *FanOutProducer {
	return &FanOutProducer{
		pusher: pusher,
	}
}

// PublishBatch 投递一个粉丝分批 以 contentID 作 key 保证同内容分批落到同分区
func (p *FanOutProducer) PublishBatch(ctx context.Context, batch *event.FanOutBatch) error {
	if p == nil || p.pusher == nil {
		return nil
	}
	if batch == nil || batch.ContentID <= 0 || len(batch.FollowerIDs) == 0 {
		return nil
	}
	body, err := batch.Marshal()
	if err != nil {
		return err
	}
	return p.pusher.PushWithKey(ctx, fmt.Sprintf("%d", batch.ContentID), body)
}
