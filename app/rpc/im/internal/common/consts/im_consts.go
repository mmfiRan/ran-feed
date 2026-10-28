package consts

import "time"

const (
	// MaxClientMsgIDLen 客户端消息标识最大字符数
	MaxClientMsgIDLen = 64
	// MaxContentLength 消息内容最大字符数,与 .api 的 max 校验和 DDL VARCHAR(4096) 对齐
	MaxContentLength = 4096
	// PushTimeout 事务提交后推送的超时,脱离请求 ctx
	PushTimeout = 5 * time.Second
)
