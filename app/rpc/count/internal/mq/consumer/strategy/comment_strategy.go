package strategy

import "ran-feed/app/rpc/count/count"

const commentTableName = "ran_feed_comment"

func init() {
	registerFactory(newCommentCanalTableStrategy)
}

func newCommentCanalTableStrategy() TableStrategy {
	return &contentCountTableStrategy{
		tableName: commentTableName,
		bizType:   count.BizType_COMMENT,
		deltaByOpFn: map[string]func(row map[string]interface{}, oldRow map[string]interface{}) int64{
			"INSERT": func(row map[string]interface{}, _ map[string]interface{}) int64 {
				if isCommentActive(row) {
					return 1
				}
				return 0
			},
			"DELETE": func(row map[string]interface{}, oldRow map[string]interface{}) int64 {
				// 物理删除：只在删除前为“活跃”时 -1，避免删除已逻辑删除的评论再次扣减
				before := map[string]interface{}{
					"status":     mergedValue(row, oldRow, "status"),
					"is_deleted": mergedValue(row, oldRow, "is_deleted"),
				}
				if isCommentActive(before) {
					return -1
				}
				return 0
			},
			"UPDATE": func(row map[string]interface{}, oldRow map[string]interface{}) int64 {
				before := map[string]interface{}{
					"status":     mergedValue(row, oldRow, "status"),
					"is_deleted": mergedValue(row, oldRow, "is_deleted"),
				}
				return boolDelta(isCommentActive(before), isCommentActive(row))
			},
		},
	}
}
