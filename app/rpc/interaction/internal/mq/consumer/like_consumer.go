package consumer

import (
	"context"

	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/entity/query"
	"ran-feed/app/rpc/interaction/internal/mq/event"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
)

type LikeConsumer struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	likeRepo     repositories.LikeRepository
	dedupRepo    repositories.MqConsumeDedupRepository
	consumerName string
}

func NewLikeConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *LikeConsumer {
	return &LikeConsumer{
		ctx:          ctx,
		svcCtx:       svcCtx,
		Logger:       logx.WithContext(ctx),
		likeRepo:     repositories.NewLikeRepository(ctx, svcCtx.MysqlDb),
		dedupRepo:    repositories.NewMqConsumeDedupRepository(ctx, svcCtx.MysqlDb),
		consumerName: "interaction.like_consumer",
	}
}

// Consume 消费点赞事件
func (c *LikeConsumer) Consume(ctx context.Context, key, val string) error {
	logc.Infof(ctx, "开始处理点赞事件: %s", val)
	likeEvent, err := event.UnmarshalLikeEvent(val)
	if err != nil {
		logc.Errorf(ctx, "解析点赞事件失败: %v, val=%s", err, val)
		return err
	}

	// 一条消息一条事务：先插入幂等记录，再更新 like 表
	return query.Q.Transaction(func(tx *query.Query) error {
		inserted, err := c.dedupRepo.WithTx(tx).InsertIfAbsent(c.consumerName, likeEvent.EventID)
		if err != nil {
			return err
		}
		if !inserted {
			logc.Infof(ctx, "事件已处理，跳过: eventId=%s", likeEvent.EventID)
			return nil
		}

		status := repositories.LikeStatusCancel
		if likeEvent.EventType == event.EventTypeLike {
			status = repositories.LikeStatusLike
		}
		likeDO := &do.LikeDO{
			UserID:        likeEvent.UserID,
			ContentID:     likeEvent.ContentID,
			ContentUserID: likeEvent.ContentUserID,
			Status:        status,
			CreatedBy:     likeEvent.UserID,
			UpdatedBy:     likeEvent.UserID,
		}
		return c.likeRepo.WithTx(tx).Upsert(likeDO)
	})
}
