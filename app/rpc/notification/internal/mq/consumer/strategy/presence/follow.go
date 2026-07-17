package presence

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logc"

	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/notification/notification"
)

const followTableName = "ran_feed_follow"

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &followStrategy{}
	})
}

type followStrategy struct{}

func (s *followStrategy) TableName() string { return followTableName }

func (s *followStrategy) ExtractEvents(ctx context.Context, op string, row, oldRow map[string]interface{}) []strategy.NotifyEvent {
	if !isActivation(op, statusActiveNotDeleted, row, oldRow) {
		return nil
	}
	actorID, ok := strategy.ParseInt64(row["user_id"])
	if !ok || actorID <= 0 {
		logc.Errorf(ctx, "follow canal 缺 user_id row=%v", row)
		return nil
	}
	recipientID, ok := strategy.ParseInt64(row["follow_user_id"])
	if !ok || recipientID <= 0 {
		logc.Errorf(ctx, "follow canal 缺 follow_user_id row=%v", row)
		return nil
	}
	if actorID == recipientID {
		return nil
	}
	// FOLLOW 走 PersistAggregate 依赖 uk_recipient_aggkey 收敛「取关→再关注」到同一行并 re-surface
	// aggKey=FO:{actor_id} agg_count 在 FOLLOW 语义无产品含义 客户端渲染忽略
	return []strategy.NotifyEvent{{
		RecipientID: recipientID,
		ActorID:     actorID,
		NotifyType:  int32(notification.NotifyType_FOLLOW),
		AggKey:      fmt.Sprintf("FO:%d", actorID),
		Action:      strategy.PersistAggregate,
	}}
}
