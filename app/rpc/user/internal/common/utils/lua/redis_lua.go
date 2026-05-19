package lua

import _ "embed"

// SaveSessionScript 登录态写入脚本
//
//go:embed save_session.lua
var SaveSessionScript string

// RemoveSessionScript 登录态删除脚本
//
//go:embed remove_session.lua
var RemoveSessionScript string

// CheckLoginRateLimitScript 登录失败滑动窗口检查脚本
//
//go:embed check_login_rate_limit.lua
var CheckLoginRateLimitScript string

// RecordLoginFailureScript 登录失败计数记录脚本
//
//go:embed record_login_failure.lua
var RecordLoginFailureScript string
