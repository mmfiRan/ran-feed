package followfeed

import _ "embed"

// updateInboxZSetScript 收件箱回填裁剪续期 返回实际新增数
//
//go:embed scripts/update_inbox_zset.lua
var updateInboxZSetScript string
