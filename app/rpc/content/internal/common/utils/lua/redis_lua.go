package lua

import _ "embed"

// QueryHotFeedZSetScript 热榜ZSET查询Lua脚本
//
//go:embed query_hot_feed_zset.lua
var QueryHotFeedZSetScript string

// RebuildHotFeedZSetScript 热榜ZSET回填/重建Lua脚本
//
//go:embed rebuild_hot_feed_zset.lua
var RebuildHotFeedZSetScript string

// MergeHotIncScript 热榜增量合并Lua脚本
//
//go:embed merge_hot_inc.lua
var MergeHotIncScript string

// FreezeHotDirtyScript 热榜脏集合冻结Lua脚本（活跃桶原子搬到处理中桶）
//
//go:embed freeze_hot_dirty.lua
var FreezeHotDirtyScript string

// RebuildHotSnapshotScript 热榜快照重建Lua脚本
//
//go:embed rebuild_hot_snapshot.lua
var RebuildHotSnapshotScript string

// UpdateFollowInboxZSetScript 关注收件箱ZSET回填/裁剪Lua脚本
//
//go:embed update_follow_inbox_zset.lua
var UpdateFollowInboxZSetScript string

// BackfillFollowInboxZSetScript 关注收件箱回填并返回实际新增数量Lua脚本
//
//go:embed backfill_follow_inbox_zset.lua
var BackfillFollowInboxZSetScript string

// QueryUserPublishZSetScript 用户发布列表ZSET查询Lua脚本
//
//go:embed query_user_publish_zset.lua
var QueryUserPublishZSetScript string

// UpdateUserPublishZSetScript 用户发布列表ZSET回填/裁剪Lua脚本
//
//go:embed update_user_publish_zset.lua
var UpdateUserPublishZSetScript string

