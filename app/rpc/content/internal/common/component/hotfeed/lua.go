// 本组件持有的 Lua 脚本 与热榜能力同进同出 不按技术归类到公共 lua 目录
//
//	query_hot_feed_zset   热榜读路径 优先快照 回退主榜
//	rebuild_hot_feed_zset 主榜写入 全量与增量共用
//	freeze_hot_dirty      脏集合双缓冲冻结 活跃桶搬处理中桶
//	rebuild_hot_snapshot  裁剪主榜并重建对外快照
package hotfeed

import _ "embed"

// queryZSetScript 热榜 zset 查询 唯一调用方是 Feed.Query
//
//go:embed scripts/query_hot_feed_zset.lua
var queryZSetScript string

// rebuildZSetScript 热榜 zset 重建 全量与增量共用
//
//go:embed scripts/rebuild_hot_feed_zset.lua
var rebuildZSetScript string

// freezeDirtyScript 热榜脏集合冻结 活跃桶原子搬到处理中桶
//
//go:embed scripts/freeze_hot_dirty.lua
var freezeDirtyScript string

// rebuildSnapshotScript 热榜快照重建
//
//go:embed scripts/rebuild_hot_snapshot.lua
var rebuildSnapshotScript string
