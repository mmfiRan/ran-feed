package presence

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logc"

	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/notification/notification"
)

const likeTableName = "ran_feed_like"

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &likeStrategy{}
	})
}

type likeStrategy struct{}

func (s *likeStrategy) TableName() string { return likeTableName }

func (s *likeStrategy) ExtractEvents(ctx context.Context, op string, row, oldRow map[string]interface{}) []strategy.NotifyEvent {
	if !isActivation(op, statusActive, row, oldRow) {
		return nil
	}
	actorID, ok := strategy.ParseInt64(row["user_id"])
	if !ok || actorID <= 0 {
		logc.Errorf(ctx, "like canal 缺 user_id row=%v", row)
		return nil
	}
	recipientID, ok := strategy.ParseInt64(row["content_user_id"])
	if !ok || recipientID <= 0 {
		logc.Errorf(ctx, "like canal 缺 content_user_id row=%v", row)
		return nil
	}
	if actorID == recipientID {
		return nil
	}
	contentID, ok := strategy.ParseInt64(row["content_id"])
	if !ok || contentID <= 0 {
		logc.Errorf(ctx, "like canal 缺 content_id row=%v", row)
		return nil
	}
	return []strategy.NotifyEvent{{
		RecipientID: recipientID,
		ActorID:     actorID,
		NotifyType:  int32(notification.NotifyType_LIKE_FAVORITE),
		AggKey:      fmt.Sprintf("LF:%d", contentID),
		Action:      strategy.PersistAggregate,
		ContentID:   contentID,
	}}
}
