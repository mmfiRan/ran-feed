package lua

import _ "embed"

// VerifyAndRenewSessionScript 登录态校验 + 续期
//
//go:embed verify_and_renew_session.lua
var VerifyAndRenewSessionScript string
