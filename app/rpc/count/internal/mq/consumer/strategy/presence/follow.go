package presence

import (
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

const followTableName = "ran_feed_follow"

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &presenceCounterStrategy{
			tableName: followTableName,
			isActive:  statusActiveNotDeleted,
			targetsOf: followTargets,
		}
	})
}