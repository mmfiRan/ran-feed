package redis

import "strconv"

const (
	// RedisFeedHotGlobalKey 全站热榜索引key feed:hot:global
	RedisFeedHotGlobalKey = "feed:hot:global"
	// RedisFeedHotGlobalRebuildKey 冷更重建影子 key feed:hot:global:rebuilding
	// 冷更先写满影子再 RENAME 覆盖主榜 重建期间主榜不空 消除对外读空窗及空结果白删
	RedisFeedHotGlobalRebuildKey = "feed:hot:global:rebuilding"
	// RedisFeedHotGlobalLatestKey 最新热榜快照key feed:hot:global:latest
	RedisFeedHotGlobalLatestKey = "feed:hot:global:latest"
	// RedisFeedHotGlobalSnapshotPrefix 热榜快照前缀 feed:hot:global:snap
	RedisFeedHotGlobalSnapshotPrefix = "feed:hot:global:snap"
	// RedisFeedHotUserSnapshotPrefix 用户热榜快照映射前缀 feed:hot:global:user
	RedisFeedHotUserSnapshotPrefix = "feed:hot:global:user"
	// RedisFeedHotGlobalIncPrefix 全站热榜增量前缀 feed:hot:global:inc 旧记账格式 已弃用 保留供过渡清理
	RedisFeedHotGlobalIncPrefix = "feed:hot:global:inc"
	// RedisFeedHotIncDefaultShards 热榜脏集合默认分片数
	RedisFeedHotIncDefaultShards = 64
	// RedisFeedHotDirtyPrefix 热榜脏集合活跃分片前缀 feed:hot:dirty 加 id 取模 shards
	// 互动只记谁脏了 Set 去重 快更回查计数总量算分
	RedisFeedHotDirtyPrefix = "feed:hot:dirty"
	// RedisFeedHotDirtyProcPrefix 热榜脏集合冻结处理前缀 feed:hot:dirty:proc 加 shard
	// 快更开始时把活跃桶 RENAME 到冻结桶 处理期间新互动安全堆进活跃桶 根治边读边写丢事件
	RedisFeedHotDirtyProcPrefix = "feed:hot:dirty:proc"
	// RedisFeedHotWriteLockPrefix 主榜写互斥锁 快更冷更共享 固定 key
	// 任一任务持锁期间另一任务拿不到锁直接退出 防止并发写同一主榜及并发删脏桶丢互动
	RedisFeedHotWriteLockPrefix = "feed:hot:global:lock:write"
	// RedisFeedHotColdPendingKey 冷更预约标志 固定 key 带 TTL
	// 冷更开工前置位 快更见到则主动让路不抢写锁 保证冷更每日必跑不被快更饿死
	// 带 TTL 防冷更异常退出未清标志导致快更被永久挡住
	RedisFeedHotColdPendingKey = "feed:hot:global:cold:pending"
	// RedisFeedHotColdLockPrefix 冷更新每日幂等锁前缀 feed:hot:global:lock:cold 加日期
	RedisFeedHotColdLockPrefix = "feed:hot:global:lock:cold"
	// RedisFeedFollowInboxPrefix 关注收件箱前缀 feed:follow:inbox
	RedisFeedFollowInboxPrefix = "feed:follow:inbox"
	// RedisFeedFollowBigVPrefix viewer 大 V 关注列表缓存前缀 feed:follow:bigv
	RedisFeedFollowBigVPrefix = "feed:follow:bigv"
	// FollowBigVEmptySentinel 已计算且为空的占位成员，避免 miss 时反复 rebuild
	FollowBigVEmptySentinel = "0"
	// RedisFeedBigVGlobalKey 全局大 V 集合 由 count 服务跨阈值维护 本服务读写关注流时命中判推拉
	RedisFeedBigVGlobalKey = "feed:bigv:global"
	// RedisFeedUserPublishPrefix 用户发布列表前缀 feed:user:publish
	RedisFeedUserPublishPrefix = "feed:user:publish"
	// RedisFeedUserFavoritePrefix 用户收藏列表前缀 feed:user:favoriteBuildUserFavoriteFeedKey
	RedisFeedUserFavoritePrefix = "feed:user:favorite"
	// RedisFeedUserFavoriteLockPrefix 用户收藏列表锁前缀 feed:user:favorite:lock
	RedisFeedUserFavoriteLockPrefix = "feed:user:favorite:lock"
	// RedisUserFavoriteFeedCapacity 用户收藏列表热头部容量 须与 interaction 侧保持一致
	RedisUserFavoriteFeedCapacity = 300
	// RedisUserFavoriteFeedExpireSeconds 用户收藏列表头部过期时间 一天 须与 interaction 侧保持一致
	RedisUserFavoriteFeedExpireSeconds = 24 * 60 * 60
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

func BuildHotFeedUserSnapshotKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedHotUserSnapshotPrefix, strconv.FormatInt(userID, 10))
}

func BuildHotFeedIncKey(shard int) string {
	return GetRedisPrefixKey(RedisFeedHotGlobalIncPrefix, strconv.Itoa(shard))
}

// BuildHotFeedDirtyKey 构造热榜脏集合活跃分片 key feed:hot:dirty 加 shard
func BuildHotFeedDirtyKey(shard int) string {
	return GetRedisPrefixKey(RedisFeedHotDirtyPrefix, strconv.Itoa(shard))
}

// BuildHotFeedDirtyProcKey 构造热榜脏集合冻结处理分片 key feed:hot:dirty:proc 加 shard
func BuildHotFeedDirtyProcKey(shard int) string {
	return GetRedisPrefixKey(RedisFeedHotDirtyProcPrefix, strconv.Itoa(shard))
}

// BuildHotFeedWriteLockKey 主榜写互斥锁 固定 key 快更冷更共享
// 整轮全程持有 防止两类任务并发写主榜及并发删脏桶丢互动
func BuildHotFeedWriteLockKey() string {
	return RedisFeedHotWriteLockPrefix
}

func BuildHotFeedColdLockKey(date string) string {
	return GetRedisPrefixKey(RedisFeedHotColdLockPrefix, date)
}

func BuildFollowInboxKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedFollowInboxPrefix, strconv.FormatInt(userID, 10))
}

func BuildFollowBigVKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedFollowBigVPrefix, strconv.FormatInt(userID, 10))
}

func BuildUserPublishFeedKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedUserPublishPrefix, strconv.FormatInt(userID, 10))
}

func BuildUserFavoriteFeedKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedUserFavoritePrefix, strconv.FormatInt(userID, 10))
}

func BuildUserFavoriteFeedLockKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedUserFavoriteLockPrefix, strconv.FormatInt(userID, 10))
}
