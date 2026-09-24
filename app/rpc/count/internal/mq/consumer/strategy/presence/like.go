package presence

import (
	countenum "ran-feed/app/rpc/count/internal/common/enums"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

const likeTableName = "ran_feed_like"

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &presenceCounterStrategy{
			tableName: likeTableName,
			isActive:  statusActive,
			targetsOf: contentTargets(countenum.BizTypeLike),
		}
	})
}
