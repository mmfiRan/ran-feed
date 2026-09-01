// Package reset 内容软删清零策略 内容被软删时把其点赞收藏评论计数级联归零
package reset

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logc"

	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
	"ran-feed/pkg/enums"
)

const contentTableName = "ran_feed_content"

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &contentResetStrategy{tableName: contentTableName}
	})
}

type contentResetStrategy struct {
	tableName string
}

func (s *contentResetStrategy) TableName() string {
	return s.tableName
}

func (s *contentResetStrategy) ExtractUpdates(ctx context.Context, op string, row map[string]interface{}, oldRow map[string]interface{}) []strategy.Update {
	if strings.ToUpper(strings.TrimSpace(op)) != "UPDATE" {
		return nil
	}
	if !isDeletedTransition(row, oldRow) {
		return nil
	}

	contentID, ok := strategy.ParseInt64(row["id"])
	if !ok || contentID <= 0 {
		logc.Errorf(ctx, "canal消息缺少有效content id table=%s row=%v", s.tableName, row)
		return nil
	}
	ownerID, _ := strategy.ParseInt64(row["user_id"])

	bizTypes := []count.BizType{count.BizType_LIKE, count.BizType_FAVORITE, count.BizType_COMMENT}
	updates := make([]strategy.Update, 0, len(bizTypes))
	for _, biz := range bizTypes {
		updates = append(updates, strategy.Update{
			BizType:    biz,
			TargetType: count.TargetType_CONTENT,
			TargetID:   contentID,
			OwnerID:    ownerID,
			Action:     strategy.UpdateActionResetToZero,
		})
	}
	return updates
}

// isDeletedTransition is_deleted 由 0 变 1 即软删瞬间
func isDeletedTransition(row, oldRow map[string]interface{}) bool {
	if row == nil || oldRow == nil {
		return false
	}
	if _, ok := oldRow["is_deleted"]; !ok {
		return false
	}
	oldVal, okOld := strategy.ParseInt64(oldRow["is_deleted"])
	newVal, okNew := strategy.ParseInt64(row["is_deleted"])
	if !okOld || !okNew {
		return false
	}
	return !enums.IsDeleted(int32(oldVal)).IsDel() && enums.IsDeleted(int32(newVal)).IsDel()
}
