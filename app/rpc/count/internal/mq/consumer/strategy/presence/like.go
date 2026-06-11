package presence

import (
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

const likeTableName = "ran_feed_like"

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &presenceCounterStrategy{
			tableName: likeTableName,
			isActive:  statusActive,
			targetsOf: contentTargets(count.BizType_LIKE),
		}
	})
}