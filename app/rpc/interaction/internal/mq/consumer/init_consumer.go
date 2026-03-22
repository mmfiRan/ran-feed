package consumer

import (
	"context"
	"ran-feed/app/rpc/interaction/internal/config"
	"ran-feed/app/rpc/interaction/internal/svc"

	"github.com/zeromicro/go-queue/kq"
	"github.com/zeromicro/go-zero/core/service"
)

func Consumers(c config.Config, ctx context.Context, svcContext *svc.ServiceContext) []service.Service {
	return []service.Service{
		kq.MustNewQueue(c.KqConsumerConf, NewLikeConsumer(ctx, svcContext)),
	}
}
