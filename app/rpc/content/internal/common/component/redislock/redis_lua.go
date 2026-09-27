package redislock

import _ "embed"

// UpdateFollowInboxZSetScript 关注收件箱ZSET回填 裁剪Lua脚本
//
//go:embed scripts/update_follow_inbox_zset.lua
var UpdateFollowInboxZSetScript string

// BackfillFollowInboxZSetScript 关注收件箱回填并返回实际新增数量Lua脚本
//
//go:embed scripts/backfill_follow_inbox_zset.lua
var BackfillFollowInboxZSetScript string

// UpdateUserPublishZSetScript 用户发布列表ZSET回填 裁剪Lua脚本
//
//go:embed scripts/update_user_publish_zset.lua
var UpdateUserPublishZSetScript string

// FilterBigVMembersScript 批量判定候选是否为全局大 V 的 Lua 脚本
//
//go:embed scripts/filter_bigv_members.lua
var FilterBigVMembersScript string
