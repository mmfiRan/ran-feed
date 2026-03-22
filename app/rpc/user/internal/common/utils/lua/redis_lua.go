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
