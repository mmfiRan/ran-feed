// Package types 审计日志查询条件与入参结构
package types

import (
	"ran-feed/app/rpc/admin/internal/entity/model"
)

// OperationLogFilter 操作日志查询条件 username/action 模糊匹配 status 精确
type OperationLogFilter struct {
	Username    string
	Action      string
	Status      int32
	StartMillis int64
	EndMillis   int64
}

// OperationLogRow 操作审计日志关联操作人用户名后的查询行
type OperationLogRow struct {
	model.RanFeedOperationLog
	Username string
}

// LoginLogFilter 登录日志查询条件 username/ip 模糊匹配
type LoginLogFilter struct {
	Username    string
	IP          string
	Status      int32
	StartMillis int64
	EndMillis   int64
}
