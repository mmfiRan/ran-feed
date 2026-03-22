package grpcx

import "google.golang.org/grpc/codes"

// gRPC 标准错误码枚举和说明
const (
	// OK 成功，无错误
	// HTTP 映射: 200 OK
	OK = codes.OK

	// Canceled 操作被取消（通常由调用者取消）
	// HTTP 映射: 499 Client Closed Request
	Canceled = codes.Canceled

	// Unknown 未知错误
	// 当错误无法映射到其他错误码时使用
	// HTTP 映射: 500 Internal Server Error
	Unknown = codes.Unknown

	// InvalidArgument 客户端指定了无效参数
	// 用于参数验证失败、格式错误等
	// HTTP 映射: 400 Bad Request
	InvalidArgument = codes.InvalidArgument

	// DeadlineExceeded 操作在完成前超时
	// 用于超过截止时间的操作
	// HTTP 映射: 504 Gateway Timeout
	DeadlineExceeded = codes.DeadlineExceeded

	// NotFound 请求的实体未找到
	// 用于资源不存在的情况
	// HTTP 映射: 404 Not Found
	NotFound = codes.NotFound

	// AlreadyExists 尝试创建的实体已存在
	// 用于创建重复资源的情况
	// HTTP 映射: 409 Conflict
	AlreadyExists = codes.AlreadyExists

	// PermissionDenied 调用者没有权限执行指定操作
	// 用于权限不足的情况（已认证但无权限）
	// HTTP 映射: 403 Forbidden
	PermissionDenied = codes.PermissionDenied

	// ResourceExhausted 资源耗尽
	// 用于配额用完、请求过多等情况
	// HTTP 映射: 429 Too Many Requests
	ResourceExhausted = codes.ResourceExhausted

	// FailedPrecondition 操作被拒绝，因为系统状态不满足前提条件
	// 用于操作顺序错误、状态不符等情况
	// 例如：删除非空目录、在未登录状态下操作
	// HTTP 映射: 400 Bad Request
	FailedPrecondition = codes.FailedPrecondition

	// Aborted 操作被中止
	// 通常是并发冲突，如读-修改-写冲突
	// HTTP 映射: 409 Conflict
	Aborted = codes.Aborted

	// OutOfRange 操作超出有效范围
	// 用于索引越界、分页参数超出范围等
	// HTTP 映射: 400 Bad Request
	OutOfRange = codes.OutOfRange

	// Unimplemented 操作未实现或不支持
	// 用于功能未实现、API 版本不支持等
	// HTTP 映射: 501 Not Implemented
	Unimplemented = codes.Unimplemented

	// Internal 内部错误
	// 服务器内部错误，通常是 bug 或系统故障
	// HTTP 映射: 500 Internal Server Error
	Internal = codes.Internal

	// Unavailable 服务当前不可用
	// 通常是临时状态，客户端可以重试
	// HTTP 映射: 503 Service Unavailable
	Unavailable = codes.Unavailable

	// DataLoss 不可恢复的数据丢失或损坏
	// 严重错误，表示数据完整性问题
	// HTTP 映射: 500 Internal Server Error
	DataLoss = codes.DataLoss

	// Unauthenticated 请求没有有效的认证凭证
	// 用于未登录、token 无效等情况
	// HTTP 映射: 401 Unauthorized
	Unauthenticated = codes.Unauthenticated
)

// gRPC 错误码说明文档
var grpcCodeDescriptions = map[codes.Code]string{
	OK:                 "成功，无错误",
	Canceled:           "操作被取消",
	Unknown:            "未知错误",
	InvalidArgument:    "请求参数无效",
	DeadlineExceeded:   "操作超时",
	NotFound:           "资源不存在",
	AlreadyExists:      "资源已存在",
	PermissionDenied:   "权限不足",
	ResourceExhausted:  "资源耗尽或请求过多",
	FailedPrecondition: "前提条件不满足",
	Aborted:            "操作被中止",
	OutOfRange:         "操作超出有效范围",
	Unimplemented:      "功能未实现",
	Internal:           "服务内部错误",
	Unavailable:        "服务不可用",
	DataLoss:           "数据丢失或损坏",
	Unauthenticated:    "未认证",
}

// GetGrpcCodeDescription 获取 gRPC 错误码的中文说明
func GetGrpcCodeDescription(code codes.Code) string {
	if desc, ok := grpcCodeDescriptions[code]; ok {
		return desc
	}
	return "未知错误码"
}
