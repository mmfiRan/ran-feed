package consts

import "time"

// FollowMaintainTimeout 关注与取关的异步维护 RPC 超时
// 已脱离请求 ctx 必须给显式上限 否则挂住的调用会永久占用 goroutine 与连接
const FollowMaintainTimeout = 5 * time.Second
