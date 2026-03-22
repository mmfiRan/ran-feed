package grpcx

import (
	"context"
	"errors"
	"ran-feed/pkg/errorx"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"
)

// FromError 将任意错误转换为gRPCStatus
func FromError(e error) error {
	if e == nil {
		return nil
	}
	// 检查是否是业务错误
	var bizErr *errorx.BizError
	if errors.As(e, &bizErr) {
		return bizErrorToGrpcStatus(bizErr)
	}

	// 检查是否已经是 gRPC Status 错误
	if _, ok := status.FromError(e); ok {
		return e
	}

	// 检查是否是context错误
	switch e {
	case context.Canceled:
		return status.Error(Canceled, "操作已取消")
	case context.DeadlineExceeded:
		return status.Error(DeadlineExceeded, "操作超时")
	}

	// 未知错误，返回兜底错误
	return status.Error(Internal, e.Error())
}

// ToError 将 gRPC Status 转换为业务错误（Client 端使用）
func ToError(ctx context.Context, e error) error {
	if e == nil {
		return nil
	}

	var bizErr *errorx.BizError
	if errors.As(e, &bizErr) {
		return e
	}
	grpcStatus, ok := status.FromError(e)
	if !ok {
		// 不是 gRPC 错误，包装为业务错误
		return errorx.New(errorx.DefaultErrorMessage, errorx.DefaultErrorCode)
	}

	return grpcStatusToBizError(ctx, grpcStatus)
}

// bizErrorToGrpcStatus将业务错误转换为gRPCStatus
// 使用 gRPC Status Details 机制传递业务错误信息
func bizErrorToGrpcStatus(bizErr *errorx.BizError) error {
	// 业务错误统一使用 Unknown 状态码，不做映射
	// 真正的业务错误码通过 Details 传递
	st := status.New(Unknown, bizErr.Message)

	// 使用 ErrorInfo 传递业务错误码
	detail := &errdetails.ErrorInfo{
		Reason: bizErr.Message,
		Metadata: map[string]string{
			"code":    strconv.FormatUint(uint64(bizErr.Code), 10),
			"message": bizErr.Message,
		},
	}

	// 添加 Details
	st, err := st.WithDetails(detail)
	if err != nil {
		// 如果添加 Details 失败，返回基础 Status
		return status.New(Unknown, bizErr.Message).Err()
	}

	return st.Err()
}

// grpcStatusToBizError 将 gRPC Status 转换为业务错误
func grpcStatusToBizError(ctx context.Context, grpcStatus *status.Status) *errorx.BizError {
	// 尝试从 Details 中提取业务错误信息
	for _, detail := range grpcStatus.Details() {
		if errInfo, ok := detail.(*errdetails.ErrorInfo); ok {
			// 从 metadata 中提取业务错误码
			if codeStr, exists := errInfo.Metadata["code"]; exists {
				// 将字符串转换为 uint32
				if code64, err := strconv.ParseUint(codeStr, 10, 32); err == nil {
					code := uint32(code64)
					message := errInfo.Metadata["message"]
					if message == "" {
						message = grpcStatus.Message()
					}
					return errorx.New(message, code)
				}
			}
		}
	}

	// 没有 Details，说明是系统异常错误，需要打印日志
	message := grpcStatus.Message()
	if message == "" {
		message = errorx.DefaultErrorMessage
	}

	// 记录系统异常错误日志
	logx.WithContext(ctx).Errorf("[gRPC Client] 收到系统异常错误: grpc_code=%s, message=%s",
		grpcStatus.Code().String(), message)

	return errorx.New(errorx.DefaultErrorMessage, errorx.DefaultErrorCode)
}
