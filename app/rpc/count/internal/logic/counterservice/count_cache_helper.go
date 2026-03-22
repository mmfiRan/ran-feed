package counterservicelogic

import (
	"fmt"
	"math/rand"
	"strconv"

	"ran-feed/app/rpc/count/count"
	rediskey "ran-feed/app/rpc/count/internal/common/consts/redis"
)

const cacheExpireJitterMaxSeconds = 600

func buildCountValueCacheKey(bizType count.BizType, targetType count.TargetType, targetID int64) string {
	return rediskey.BuildCountValueKey(
		strconv.FormatInt(int64(bizType), 10),
		strconv.FormatInt(int64(targetType), 10),
		strconv.FormatInt(targetID, 10),
	)
}

func buildCountValueMapKey(bizType count.BizType, targetType count.TargetType, targetID int64) string {
	return fmt.Sprintf("%d:%d:%d", bizType, targetType, targetID)
}

func countCacheExpireSecondsWithJitter() int {
	return rediskey.RedisCountValueExpireSeconds + rand.Intn(cacheExpireJitterMaxSeconds+1)
}

func buildUserProfileCountsCacheKey(userID int64) string {
	return rediskey.GetRedisPrefixKey(rediskey.RedisUserProfileCountsPrefix, strconv.FormatInt(userID, 10))
}
