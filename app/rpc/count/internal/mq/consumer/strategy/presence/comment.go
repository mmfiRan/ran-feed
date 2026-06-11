package presence

import (
	"ran-feed/app/rpc/count/count"
	"ran-feed/app/rpc/count/internal/mq/consumer/strategy"
)

const commentTableName = "ran_feed_comment"

func init() {
	strategy.RegisterFactory(func() strategy.TableStrategy {
		return &presenceCounterStrategy{
			tableName: commentTableName,
			isActive:  statusActiveNotDeleted,
			targetsOf: contentTargets(count.BizType_COMMENT),
		}
	})
}