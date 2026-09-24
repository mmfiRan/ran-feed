package presence

import (
	countenum "ran-feed/app/rpc/count/internal/common/enums"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

const favoriteTableName = "ran_feed_favorite"

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &presenceCounterStrategy{
			tableName: favoriteTableName,
			isActive:  alwaysActive,
			targetsOf: contentTargets(countenum.BizTypeFavorite),
		}
	})
}
