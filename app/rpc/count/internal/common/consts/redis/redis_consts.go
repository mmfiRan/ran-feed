package redis

import "strconv"

const (
	// RedisCountValuePrefix 统一计数缓存前缀 count:value:{biz_type}:{target_type}:{target_id}
	RedisCountValuePrefix = "count:value"
	// RedisCountValueExpireSeconds 计数缓存过期时间：24小时
	RedisCountValueExpireSeconds = 24 * 60 * 60
	// RedisCountRebuildLockPrefix 计数缓存重建锁前缀 lock:rebuild:count:{biz_type}:{target_type}:{target_id}
	RedisCountRebuildLockPrefix = "lock:rebuild:count"
	// RedisUserProfileCountsPrefix 用户主页计数缓存前缀 count:user:profile:{user_id}
	RedisUserProfileCountsPrefix = "count:user:profile"
	// RedisUserProfileCountsRebuildLockPrefix 用户主页计数重建锁前缀 lock:rebuild:count:user:profile:{user_id}
	RedisUserProfileCountsRebuildLockPrefix = "lock:rebuild:count:user:profile"
	// RedisFeedHotGlobalIncPrefix 热榜增量分片前缀（与 content 服务热榜任务约定一致）
	RedisFeedHotGlobalIncPrefix = "feed:hot:global:inc"
	// RedisFeedHotIncDefaultShards 热榜增量默认分片数
	RedisFeedHotIncDefaultShards = 64
)

func GetRedisPrefixKey(prefix string, id string) string {
	return prefix + ":" + id
}

func BuildCountValueKey(bizType string, targetType string, targetID string) string {
	return GetRedisPrefixKey(GetRedisPrefixKey(GetRedisPrefixKey(RedisCountValuePrefix, bizType), targetType), targetID)
}

func BuildHotFeedIncKey(shard int) string {
	return GetRedisPrefixKey(RedisFeedHotGlobalIncPrefix, strconv.Itoa(shard))
}
