package producer

import (
	"context"
	"fmt"
	"time"

	"ran-feed/app/rpc/interaction/internal/mq/event"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/logx"
)

// LikeProducer 点赞事件生产者
type LikeProducer struct {
	pusher     *kq.Pusher
	maxRetries int // 最大重试次数
}

func NewLikeProducer(pusher *kq.Pusher, maxRetries int) *LikeProducer {
	return &LikeProducer{
		pusher:     pusher,
		maxRetries: maxRetries,
	}
}

// SendLikeEvent 发送点赞事件
func (p *LikeProducer) SendLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) {
	likeEvent := &event.LikeEvent{
		EventID:       fmt.Sprintf("like_%d_%d_%d", userID, contentID, time.Now().UnixNano()),
		EventType:     event.EventTypeLike,
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Scene:         scene,
		Timestamp:     time.Now().Unix(),
	}
	p.sendEventWithRetry(ctx, likeEvent)
}

// SendCancelLikeEvent 发送取消点赞事件
func (p *LikeProducer) SendCancelLikeEvent(ctx context.Context, userID, contentID, contentUserID int64, scene string) {
	likeEvent := &event.LikeEvent{
		EventID:       fmt.Sprintf("cancel_like_%d_%d_%d", userID, contentID, time.Now().UnixNano()),
		EventType:     event.EventTypeCancel,
		UserID:        userID,
		ContentID:     contentID,
		ContentUserID: contentUserID,
		Scene:         scene,
		Timestamp:     time.Now().Unix(),
	}
	p.sendEventWithRetry(ctx, likeEvent)
}

// sendEventWithRetry 带重试的发送事件
func (p *LikeProducer) sendEventWithRetry(ctx context.Context, evt *event.LikeEvent) {
	var lastErr error

	for i := 0; i < p.maxRetries; i++ {
		if err := p.sendEvent(ctx, evt); err == nil {
			return
		} else {
			lastErr = err
			logx.WithContext(ctx).Errorf("发送事件失败，重试 %d次 %d: %v", i+1, p.maxRetries, err)
			time.Sleep(time.Millisecond * 100 * time.Duration(i+1)) // 指数退避
		}
	}

	if lastErr != nil {
		logx.WithContext(ctx).Errorf("发送事件失败，达到最大重试次数 eventId=%s, error=%v, 原始数据=%v", evt.EventID, lastErr, evt)

	}
}

// sendEvent 发送事件到Kafka
func (p *LikeProducer) sendEvent(ctx context.Context, evt *event.LikeEvent) error {
	body, err := evt.Marshal()
	if err != nil {
		logx.WithContext(ctx).Errorf("序列化点赞事件失败: %v", err)
		return err
	}

	if err = p.pusher.Push(ctx, body); err != nil {
		logx.WithContext(ctx).Errorf("发送点赞事件到Kafka失败: %v, event=%+v", err, evt)
		return err
	}

	logx.WithContext(ctx).Infof("发送点赞事件成功: eventId=%s, type=%s", evt.EventID, evt.EventType)
	return nil
}
