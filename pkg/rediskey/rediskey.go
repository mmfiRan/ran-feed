// Package rediskey 跨服务共享的 Redis key
// 判据是两个服务读到不同的 Prefix 会静默出错 这类 key 只能有一处定义
// 只放能说出业务对象的 Builder 不放通用字符串拼接函数
package rediskey

import "strconv"

const (
	// FeedBigVGlobal 全局大 V 集合 feed:bigv:global
	// 粉丝数跨阈值由计数域增量维护 内容域读写关注流时命中判推拉
	FeedBigVGlobal = "feed:bigv:global"
	// FeedBigVGlobalRebuild 大 V 集合周期重建临时 key 建好后 RENAME 原子换到正式 key
	FeedBigVGlobalRebuild = "feed:bigv:global:rebuild"
	// feedHotDirtyPrefix 热榜脏集合活跃分片前缀 互动方只记谁脏了 内容域快更回查计数算分
	feedHotDirtyPrefix = "feed:hot:dirty"
)

// HotFeedDirty 热榜脏集合活跃分片 key feed:hot:dirty 加 target_id 取模的 shard
// 计数域写 内容域读 两侧分片数必须同值
func HotFeedDirty(shard int) string {
	return feedHotDirtyPrefix + ":" + strconv.Itoa(shard)
}
