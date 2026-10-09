// Package rediskey 跨服务共享的 Redis key
package rediskey

import "strconv"

const (
	// FeedBigVGlobal 全局大 V 集合
	FeedBigVGlobal = "feed:bigv:global"
	// FeedBigVGlobalRebuild 大 V 集合周期重建临时 key 建好后 RENAME 原子换到正式 key
	FeedBigVGlobalRebuild = "feed:bigv:global:rebuild"
	// feedHotDirtyPrefix 热榜脏集合活跃分片前缀 互动方只记谁脏了
	feedHotDirtyPrefix = "feed:hot:dirty"
)

// HotFeedDirty 热榜脏集合活跃分片 key feed:hot:dirty
func HotFeedDirty(shard int) string {
	return feedHotDirtyPrefix + ":" + strconv.Itoa(shard)
}
