package presence

import (
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

const favoriteTableName = "ran_feed_favorite"

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &presenceCounterStrategy{
			tableName: favoriteTableName,
			isActive:  alwaysActive,
			targetsOf: contentTargets(count.BizType_BIZ_TYPE_FAVORITE),
		}
	})
}
