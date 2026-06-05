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
	// RedisFeedHotGlobalIncPrefix 热榜增量分片前缀 旧记账格式 已弃用 保留供过渡清理
	RedisFeedHotGlobalIncPrefix = "feed:hot:global:inc"
	// RedisFeedHotIncDefaultShards 热榜脏集合默认分片数 与 content 服务热榜任务约定一致
	RedisFeedHotIncDefaultShards = 64
	// RedisFeedHotDirtyPrefix 热榜脏集合活跃分片前缀 feed:hot:dirty 加 id 取模 shards
	// 互动只记谁脏了 Set 去重 算分由 content 快更任务回查计数总量批量完成
	RedisFeedHotDirtyPrefix = "feed:hot:dirty"
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

// BuildHotFeedDirtyKey 构造热榜脏集合分片 key feed:hot:dirty 加 shard
func BuildHotFeedDirtyKey(shard int) string {
	return GetRedisPrefixKey(RedisFeedHotDirtyPrefix, strconv.Itoa(shard))
}
