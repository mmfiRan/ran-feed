package strategy

import "ran-feed/app/rpc/count/count"

const likeTableName = "ran_feed_like"

func init() {
	registerFactory(newLikeCanalTableStrategy)
}

func newLikeCanalTableStrategy() TableStrategy {
	return &contentCountTableStrategy{
		tableName: likeTableName,
		bizType:   count.BizType_LIKE,
		deltaByOpFn: map[string]func(row map[string]interface{}, oldRow map[string]interface{}) int64{
			"INSERT": func(row map[string]interface{}, _ map[string]interface{}) int64 {
				if isLikeActive(row) {
					return 1
				}
				return 0
			},
			"DELETE": func(row map[string]interface{}, _ map[string]interface{}) int64 {
				if isLikeActive(row) {
					return -1
				}
				return 0
			},
			"UPDATE": func(row map[string]interface{}, oldRow map[string]interface{}) int64 {
				before := map[string]interface{}{
					"status": mergedValue(row, oldRow, "status"),
				}
				return boolDelta(isLikeActive(before), isLikeActive(row))
			},
		},
	}
}
