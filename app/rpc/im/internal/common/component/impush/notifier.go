// Package impush im:push channel 发布的唯一 owner
package impush

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/stores/redis"

	"ran-feed/pkg/event/imevent"
	"ran-feed/pkg/rediskey"
)

// Notifier im 实时下行发布器
type Notifier struct {
	redis *redis.Redis
}

// New 按 Redis 配置建发布客户端
func New(conf redis.RedisConf) *Notifier {
	return &Notifier{redis: redis.MustNewRedis(conf)}
}

// Notify 把帧投递到 im:push,由 front-api 各 pod 订阅后按用户扇出
func (n *Notifier) Notify(ctx context.Context, userIDs []int64, frame *imevent.Frame) error {
	if n == nil || len(userIDs) == 0 || frame == nil {
		return nil
	}
	frameBytes, err := json.Marshal(frame)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(imevent.Envelope{UserIDs: userIDs, Frame: frameBytes})
	if err != nil {
		return err
	}
	_, err = n.redis.PublishCtx(ctx, rediskey.IMPushChannel, string(payload))
	return err
}
