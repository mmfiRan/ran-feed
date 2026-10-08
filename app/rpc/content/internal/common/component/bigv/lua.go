package bigv

import _ "embed"

// filterMembersScript 批量判定候选是否为全局大 V 唯一调用方是 Set.Filter
//
//go:embed scripts/filter_members.lua
var filterMembersScript string
