package strategy

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logc"

	"ran-feed/app/rpc/count/count"
)

const followTableName = "ran_feed_follow"

func init() {
	registerFactory(newFollowCanalTableStrategy)
}

func newFollowCanalTableStrategy() TableStrategy {
	return &followCountTableStrategy{
		tableName: followTableName,
		deltaByOpFn: map[string]func(row map[string]interface{}, oldRow map[string]interface{}) int64{
			"INSERT": func(row map[string]interface{}, _ map[string]interface{}) int64 {
				if isFollowActive(row) {
					return 1
				}
				return 0
			},
			"DELETE": func(row map[string]interface{}, _ map[string]interface{}) int64 {
				if isFollowActive(row) {
					return -1
				}
				return 0
			},
			"UPDATE": func(row map[string]interface{}, oldRow map[string]interface{}) int64 {
				before := map[string]interface{}{
					"status":     mergedValue(row, oldRow, "status"),
					"is_deleted": mergedValue(row, oldRow, "is_deleted"),
				}
				return boolDelta(isFollowActive(before), isFollowActive(row))
			},
		},
	}
}

type followCountTableStrategy struct {
	tableName   string
	deltaByOpFn map[string]func(row map[string]interface{}, oldRow map[string]interface{}) int64
}

func (s *followCountTableStrategy) TableName() string {
	return s.tableName
}

func (s *followCountTableStrategy) ExtractUpdates(ctx context.Context, op string, row map[string]interface{}, oldRow map[string]interface{}) []Update {
	handler, ok := s.deltaByOpFn[strings.ToUpper(strings.TrimSpace(op))]
	if !ok || handler == nil {
		return nil
	}

	userID, ok := getInt64(row["user_id"])
	if !ok || userID <= 0 {
		logc.Errorf(ctx, "canal消息缺少有效user_id: table=%s, op=%s, row=%v", s.tableName, op, row)
		return nil
	}
	followUserID, ok := getInt64(row["follow_user_id"])
	if !ok || followUserID <= 0 {
		logc.Errorf(ctx, "canal消息缺少有效follow_user_id: table=%s, op=%s, row=%v", s.tableName, op, row)
		return nil
	}

	delta := handler(row, oldRow)
	if delta == 0 {
		return nil
	}

	return []Update{
		{
			BizType:    count.BizType_FOLLOWING,
			TargetType: count.TargetType_USER,
			TargetID:   userID,
			Delta:      delta,
		},
		{
			BizType:    count.BizType_FOLLOWED,
			TargetType: count.TargetType_USER,
			TargetID:   followUserID,
			Delta:      delta,
		},
	}
}
