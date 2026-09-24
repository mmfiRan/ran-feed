package consumer

import (
	"context"

	contentconsts "ran-feed/app/rpc/content/internal/common/consts"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/common/utils/followwindow"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/mq/event"
	"ran-feed/app/rpc/content/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// FanOutConsumer 消费扇出分批消息 逐批把粉丝写进各自收件箱 独立 group 与在线读隔离
type FanOutConsumer struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFanOutConsumer(ctx context.Context, svcCtx *svc.ServiceContext) *FanOutConsumer {
	return &FanOutConsumer{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Consume 整批一次 pipeline 写收件箱 任一步失败整批返回错误交 kafka 重投 写收件箱幂等可重放
func (c *FanOutConsumer) Consume(ctx context.Context, key, val string) error {
	batch, err := event.UnmarshalFanOutBatch(val)
	if err != nil {
		// 报文损坏重投也无法成功 记日志跳过 不阻塞分区
		c.Errorf("解析扇出消息失败 跳过 err=%v val=%s", err, val)
		return nil
	}
	if batch == nil || batch.ContentID <= 0 || batch.AuthorID <= 0 || len(batch.FollowerIDs) == 0 {
		return nil
	}

	days := contentconsts.WindowDays
	inboxArgs := followwindow.WriteArgs(contentconsts.TimelineKeepN, followwindow.CutoffMillis(days), followwindow.TTLSeconds(days), batch.PublishedAt, batch.ContentID)

	return c.svcCtx.Redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		for _, followerID := range batch.FollowerIDs {
			if followerID <= 0 {
				continue
			}
			pipe.Eval(ctx, luautils.UpdateFollowInboxZSetScript, []string{rediskey.BuildFollowInboxKey(followerID)}, inboxArgs...)
		}
		return nil
	})
}
