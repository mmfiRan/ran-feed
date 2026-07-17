package presence

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logc"

	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/notification/notification"
)

const favoriteTableName = "ran_feed_favorite"

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &favoriteStrategy{}
	})
}

type favoriteStrategy struct{}

func (s *favoriteStrategy) TableName() string { return favoriteTableName }

func (s *favoriteStrategy) ExtractEvents(ctx context.Context, op string, row, oldRow map[string]interface{}) []strategy.NotifyEvent {
	// favorite 硬删无 status 列 alwaysActive 即 INSERT/UPDATE 视为激活
	// DELETE 分支 isActivation 返 false 不产通知(取消收藏不打扰)
	if !isActivation(op, alwaysActive, row, oldRow) {
		return nil
	}
	actorID, ok := strategy.ParseInt64(row["user_id"])
	if !ok || actorID <= 0 {
		logc.Errorf(ctx, "favorite canal 缺 user_id row=%v", row)
		return nil
	}
	recipientID, ok := strategy.ParseInt64(row["content_user_id"])
	if !ok || recipientID <= 0 {
		logc.Errorf(ctx, "favorite canal 缺 content_user_id row=%v", row)
		return nil
	}
	if actorID == recipientID {
		return nil
	}
	contentID, ok := strategy.ParseInt64(row["content_id"])
	if !ok || contentID <= 0 {
		logc.Errorf(ctx, "favorite canal 缺 content_id row=%v", row)
		return nil
	}
	// 与 like 同一 aggKey LF:{content_id} 使赞与收藏合并聚合(N5)
	return []strategy.NotifyEvent{{
		RecipientID: recipientID,
		ActorID:     actorID,
		NotifyType:  int32(notification.NotifyType_LIKE_FAVORITE),
		AggKey:      fmt.Sprintf("LF:%d", contentID),
		Action:      strategy.PersistAggregate,
		ContentID:   contentID,
	}}
}
