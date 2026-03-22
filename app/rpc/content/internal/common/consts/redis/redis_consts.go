package redis

import "strconv"

const (
	// RedisFeedHotGlobalKey 全站热榜索引key feed:hot:global
	RedisFeedHotGlobalKey = "feed:hot:global"
	// RedisFeedHotGlobalLatestKey 最新热榜快照key feed:hot:global:latest
	RedisFeedHotGlobalLatestKey = "feed:hot:global:latest"
	// RedisFeedHotGlobalSnapshotPrefix 热榜快照前缀 feed:hot:global:snap
	RedisFeedHotGlobalSnapshotPrefix = "feed:hot:global:snap"
	// RedisFeedHotUserSnapshotPrefix 用户热榜快照映射前缀 feed:hot:global:user
	RedisFeedHotUserSnapshotPrefix = "feed:hot:global:user"
	// RedisFeedHotGlobalIncPrefix 全站热榜增量前缀 feed:hot:global:inc
	RedisFeedHotGlobalIncPrefix = "feed:hot:global:inc"
	// RedisFeedHotIncDefaultShards 热榜增量默认分片数
	RedisFeedHotIncDefaultShards = 64
	// RedisFeedHotFastLockPrefix 快速更新锁前缀 feed:hot:global:lock:fast
	RedisFeedHotFastLockPrefix = "feed:hot:global:lock:fast"
	// RedisFeedHotColdLockPrefix 冷更新锁前缀 feed:hot:global:lock:cold
	RedisFeedHotColdLockPrefix = "feed:hot:global:lock:cold"
	// RedisFeedFollowInboxPrefix 关注收件箱前缀 feed:follow:inbox
	RedisFeedFollowInboxPrefix = "feed:follow:inbox"
	// RedisFeedFollowInboxRebuildLockPrefix 关注收件箱重建锁前缀 feed:follow:inbox:lock
	RedisFeedFollowInboxRebuildLockPrefix = "feed:follow:inbox:lock"
	// RedisFeedUserPublishPrefix 用户发布列表前缀 feed:user:publish
	RedisFeedUserPublishPrefix = "feed:user:publish"
	// RedisFeedUserFavoritePrefix 用户收藏列表前缀 feed:user:favoriteBuildUserFavoriteFeedKey
	RedisFeedUserFavoritePrefix = "feed:user:favorite"
	// RedisFeedUserFavoriteLockPrefix 用户收藏列表锁前缀 feed:user:favorite:lock
	RedisFeedUserFavoriteLockPrefix = "feed:user:favorite:lock"
)

func GetRedisPrefixKey(prefix string, id string) string {
	return prefix + ":" + id
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

func BuildHotFeedFastLockKey(bucket string) string {
	return GetRedisPrefixKey(RedisFeedHotFastLockPrefix, bucket)
}

func BuildHotFeedColdLockKey(date string) string {
	return GetRedisPrefixKey(RedisFeedHotColdLockPrefix, date)
}

func BuildFollowInboxKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedFollowInboxPrefix, strconv.FormatInt(userID, 10))
}

func BuildFollowInboxRebuildLockKey(userID int64) string {
	return GetRedisPrefixKey(RedisFeedFollowInboxRebuildLockPrefix, strconv.FormatInt(userID, 10))
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
