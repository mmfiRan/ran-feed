package consumer

import (
	"context"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"

	"ran-feed/app/rpc/notification/internal/config"
	"ran-feed/app/rpc/notification/internal/svc"
)

// Consumers 装配 Canal 消费者 topic 未配置视为未启用消费(骨架 / 单测场景)
func Consumers(c config.Config, ctx context.Context, svcContext *svc.ServiceContext) []service.Service {
	consumers := make([]service.Service, 0)
	if c.KqConsumerConf.Topic != "" {
		consumers = append(consumers, kq.MustNewQueue(c.KqConsumerConf, NewCanalNotificationConsumer(ctx, svcContext)))
	}
	return consumers
}
