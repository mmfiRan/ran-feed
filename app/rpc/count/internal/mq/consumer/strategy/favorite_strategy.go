package strategy

import "ran-feed/app/rpc/count/count"

const favoriteTableName = "ran_feed_favorite"

func init() {
	registerFactory(newFavoriteCanalTableStrategy)
}

func newFavoriteCanalTableStrategy() TableStrategy {
	return &contentCountTableStrategy{
		tableName: favoriteTableName,
		bizType:   count.BizType_FAVORITE,
		deltaByOpFn: map[string]func(row map[string]interface{}, oldRow map[string]interface{}) int64{
			"INSERT": func(map[string]interface{}, map[string]interface{}) int64 { return 1 },
			"DELETE": func(map[string]interface{}, map[string]interface{}) int64 { return -1 },
			"UPDATE": func(map[string]interface{}, map[string]interface{}) int64 { return 0 },
		},
	}
}
