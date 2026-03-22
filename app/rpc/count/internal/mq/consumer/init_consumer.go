package consumer

import (
	"context"
	"ran-feed/app/rpc/count/internal/config"
	"ran-feed/app/rpc/count/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
)

func Consumers(c config.Config, ctx context.Context, svcContext *svc.ServiceContext) []service.Service {
	consumers := make([]service.Service, 0)
	if c.KqConsumerConf.Topic != "" {
		consumers = append(consumers, kq.MustNewQueue(c.KqConsumerConf, NewCanalCountConsumer(ctx, svcContext)))
	}
	return consumers
}
