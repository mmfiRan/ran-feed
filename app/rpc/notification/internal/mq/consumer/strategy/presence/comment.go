package presence

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logc"

	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
	"ran-feed/app/rpc/notification/notification"
)

const (
	commentTableName = "ran_feed_comment"
	// snippetMaxRunes 与 ran_feed_notification.snippet varchar(140) 对齐 按字符不按字节截断避免中文半字节
	snippetMaxRunes = 140
)

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &commentStrategy{}
	})
}

type commentStrategy struct{}

func (s *commentStrategy) TableName() string { return commentTableName }

func (s *commentStrategy) ExtractEvents(ctx context.Context, op string, row, oldRow map[string]interface{}) []strategy.NotifyEvent {
	if !isActivation(op, statusActiveNotDeleted, row, oldRow) {
		return nil
	}
	actorID, ok := strategy.ParseInt64(row["user_id"])
	if !ok || actorID <= 0 {
		logc.Errorf(ctx, "comment canal 缺 user_id row=%v", row)
		return nil
	}
	contentID, ok := strategy.ParseInt64(row["content_id"])
	if !ok || contentID <= 0 {
		logc.Errorf(ctx, "comment canal 缺 content_id row=%v", row)
		return nil
	}
	commentID, ok := strategy.ParseInt64(row["id"])
	if !ok || commentID <= 0 {
		logc.Errorf(ctx, "comment canal 缺 id row=%v", row)
		return nil
	}
	parentID, _ := strategy.ParseInt64(row["parent_id"])

	// 按 parent_id 分流 recipient(N5):顶评→内容作者 回复→父评论作者
	var recipientID int64
	if parentID == 0 {
		recipientID, _ = strategy.ParseInt64(row["content_user_id"])
	} else {
		recipientID, _ = strategy.ParseInt64(row["reply_to_user_id"])
	}
	if recipientID <= 0 || actorID == recipientID {
		// 自我评论/回复不打扰(N10)
		return nil
	}

	snippet := truncateRunes(strategy.ParseString(row["comment"]), snippetMaxRunes)

	return []strategy.NotifyEvent{{
		RecipientID: recipientID,
		ActorID:     actorID,
		NotifyType:  int32(notification.NotifyType_COMMENT_REPLY),
		AggKey:      fmt.Sprintf("CR:%d", commentID),
		Action:      strategy.PersistInsertOne,
		ContentID:   contentID,
		CommentID:   commentID,
		Snippet:     snippet,
	}}
}

// truncateRunes 按 rune 截断避免中文半字节
func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
