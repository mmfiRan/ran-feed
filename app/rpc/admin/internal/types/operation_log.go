// Package types 审计日志查询条件与入参结构
package types

// OperationLogFilter 操作日志查询条件
type OperationLogFilter struct {
	AdminID     int64
	Action      string
	Status      int32
	StartMillis int64
	EndMillis   int64
}

// LoginLogFilter 登录日志查询条件 username/ip 模糊匹配
type LoginLogFilter struct {
	Username    string
	IP          string
	Status      int32
	StartMillis int64
	EndMillis   int64
}
