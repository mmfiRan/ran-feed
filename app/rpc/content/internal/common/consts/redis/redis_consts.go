package redis

import "strconv"

const (
	// RedisFeedHotGlobalKey 全站热榜索引key feed:hot:global
	RedisFeedHotGlobalKey = "feed:hot:global"
	// RedisFeedHotGlobalLatestKey 最新热榜快照key feed:hot:global:latest
	RedisFeedHotGlobalLatestKey = "feed:hot:global:latest"
	// RedisFeedHotGlobalSnapshotPrefix 热榜快照前缀 feed:hot:global:snap
	RedisFeedHotGlobalSnapshotPrefix = "feed:hot:global:snap"
	// RedisFeedHotDirtyPrefix 热榜脏集合活跃分片前缀 feed:hot:dirty 加 id 取模 shards
	// 互动只记谁脏了 Set 去重 快更回查计数总量算分
	RedisFeedHotDirtyPrefix = "feed:hot:dirty"
	// RedisFeedHotDirtyProcPrefix 热榜脏集合冻结处理前缀 feed:hot:dirty:proc 加 shard
	// 快更开始时把活跃桶 RENAME 到冻结桶 处理期间新互动安全堆进活跃桶 根治边读边写丢事件
	RedisFeedHotDirtyProcPrefix = "feed:hot:dirty:proc"
	// RedisFeedHotFullLockPrefix 全量重建幂等锁 固定 key 仅防多实例同时全量重建 不与增量互斥
	RedisFeedHotFullLockPrefix = "feed:hot:global:lock:full"
	// RedisFeedFollowInboxPrefix 关注收件箱前缀 feed:follow:inbox
	RedisFeedFollowInboxPrefix = "feed:follow:inbox"
	// RedisFeedFollowPullPrefix 拉模式关注集前缀 feed:follow:pull 装 viewer 关注里走拉模式的作者
	// 与收件箱同一次重建划分出的两半 同一 TTL 同生同死 保证推拉并集恒等于关注列表
	RedisFeedFollowPullPrefix = "feed:follow:pull"
	// FollowPullEmptySentinel 已计算且为空的占位成员 避免 miss 时反复 rebuild
	FollowPullEmptySentinel = "0"
	// RedisFeedBigVGlobalKey 全局大 V 集合 由 count 服务跨阈值维护 本服务读写关注流时命中判推拉
	RedisFeedBigVGlobalKey = "feed:bigv:global"
	// RedisFeedUserPublishPrefix 用户发布列表前缀 feed:user:publish
	RedisFeedUserPublishPrefix = "feed:user:publish"
	// RedisFeedUserFavoritePrefix 用户收藏列表前缀 feed:user:favorite
	RedisFeedUserFavoritePrefix = "feed:user:favorite"
	// RedisContentDetailPrefix feed 二级缓存内容详情前缀 content:detail 加 content_id 全 feed 共享
	RedisContentDetailPrefix = "content:detail"
	// RedisContentDetailMissingSentinel 内容详情负哨兵 已删/未发布回源查不到时写入防穿透
	RedisContentDetailMissingSentinel = "-"
)

func GetRedisPrefixKey(prefix string, id string) string {
	return prefix + ":" + id
}

// BuildContentDetailKey 构造内容详情二级缓存 key content:detail 加 content_id
func BuildContentDetailKey(contentID int64) string {
	return GetRedisPrefixKey(RedisContentDetailPrefix, strconv.FormatInt(contentID, 10))
}

func BuildHotFeedSnapshotKey(snapshotID string) string {
	return GetRedisPrefixKey(RedisFeedHotGlobalSnapshotPrefix, snapshotID)
}

// BuildHotFeedDirtyKey 构造热榜脏集合活跃分片 key feed:hot:dirty 加 shard
func BuildHotFeedDirtyKey(shard int) string {
	return GetRedisPrefixKey(RedisFeedHotDirtyPrefix, strconv.Itoa(shard))
}

// BuildHotFeedDirtyProcKey 构造热榜脏集合冻结处理分片 key feed:hot:dirty:proc 加 shard
func BuildHotFeedDirtyProcKey(shard int) string {
	return GetRedisPrefixKey(RedisFeedHotDirtyProcPrefix, strconv.Itoa(shard))
}

// BuildHotFeedFullLockKey 全量重建幂等锁 固定 key
func BuildHotFeedFullLockKey() string {
	return RedisFeedHotFullLockPrefix
}

func BuildFollowInboxKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedFollowInboxPrefix, strconv.FormatInt(userID, 10))
}

func BuildFollowPullKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedFollowPullPrefix, strconv.FormatInt(userID, 10))
}

func BuildUserPublishFeedKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedUserPublishPrefix, strconv.FormatInt(userID, 10))
}

func BuildUserFavoriteFeedKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedUserFavoritePrefix, strconv.FormatInt(userID, 10))
}
