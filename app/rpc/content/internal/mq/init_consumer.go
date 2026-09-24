package mq

import (
	"context"
	"ran-feed/app/rpc/content/internal/mq/consumer"

	"ran-feed/app/rpc/content/internal/config"
	"ran-feed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
)

func Consumers(c config.Config, ctx context.Context, svcContext *svc.ServiceContext) []service.Service {
	consumers := make([]service.Service, 0)
	if c.KqConsumerConf.Topic != "" {
		consumers = append(consumers, kq.MustNewQueue(c.KqConsumerConf, consumer.NewContentEventConsumer(ctx, svcContext)))
	}
	if c.KqFanOutConsumerConf.Topic != "" {
		consumers = append(consumers, kq.MustNewQueue(c.KqFanOutConsumerConf, consumer.NewFanOutConsumer(ctx, svcContext)))
	}
	return consumers
}
