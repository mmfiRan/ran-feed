package publishbox

import _ "embed"

// updateZSetScript 发件箱 zset 回填与裁剪 唯一调用方是本包的 write
//
//go:embed scripts/update_zset.lua
var updateZSetScript string
