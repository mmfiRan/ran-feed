package strategy

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logc"

	"ran-feed/app/rpc/count/count"
)

const contentTableName = "ran_feed_content"

func init() {
	registerFactory(newContentDeleteCanalTableStrategy)
}

type contentDeleteStrategy struct {
	tableName string
}

func newContentDeleteCanalTableStrategy() TableStrategy {
	return &contentDeleteStrategy{
		tableName: contentTableName,
	}
}

func (s *contentDeleteStrategy) TableName() string {
	return s.tableName
}

func (s *contentDeleteStrategy) ExtractUpdates(ctx context.Context, op string, row map[string]interface{}, oldRow map[string]interface{}) []Update {
	if strings.ToUpper(strings.TrimSpace(op)) != "UPDATE" {
		return nil
	}
	if !isContentDeletedTransition(row, oldRow) {
		return nil
	}

	contentID, ok := getInt64(row["id"])
	if !ok || contentID <= 0 {
		logc.Errorf(ctx, "canal消息缺少有效content id: table=%s, op=%s, row=%v", s.tableName, op, row)
		return nil
	}
	ownerID, _ := getInt64(row["user_id"])

	return []Update{
		{
			BizType:    count.BizType_LIKE,
			TargetType: count.TargetType_CONTENT,
			TargetID:   contentID,
			OwnerID:    ownerID,
			Action:     UpdateActionResetToZero,
		},
		{
			BizType:    count.BizType_FAVORITE,
			TargetType: count.TargetType_CONTENT,
			TargetID:   contentID,
			OwnerID:    ownerID,
			Action:     UpdateActionResetToZero,
		},
		{
			BizType:    count.BizType_COMMENT,
			TargetType: count.TargetType_CONTENT,
			TargetID:   contentID,
			OwnerID:    ownerID,
			Action:     UpdateActionResetToZero,
		},
	}
}
