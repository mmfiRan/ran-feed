package result

import (
	"context"
	"errors"
	"net/http"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/validate"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	Success = "success"
)

type Result struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func NewSuccessResult(data any) Result {
	return Result{
		Code:    http.StatusOK,
		Message: Success,
		Data:    data,
	}
}

func SetCustomSuccessResult(ctx context.Context, data any) any {
	return NewSuccessResult(data)
}

func NewErrorResult(code uint32, message string) Result {
	return Result{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

func SetCustomErrorResult(ctx context.Context, err error) (int, any) {
	var ce *errorx.BizError
	var cv *validate.CustomValidator

	switch {
	case errors.As(err, &ce):
		// 业务可恢复错误：HTTP 仍 200，前端按 body code 处理
		logx.WithContext(ctx).Errorf("business error: %s", err.Error())
		return http.StatusOK, NewErrorResult(ce.Code, ce.Message)
	case errors.As(err, &cv):
		// 参数校验失败本质属于"用户输入"业务错误：HTTP 仍 200，body 带 message
		logx.WithContext(ctx).Errorf("validation error: %s", err.Error())
		return http.StatusOK, NewErrorResult(errorx.DefaultErrorCode, cv.Error())
	default:
		// 非预期系统错误：HTTP 500，让监控/告警能识别
		logx.WithContext(ctx).Errorf("unknown error: %s", err.Error())
		return http.StatusInternalServerError, NewErrorResult(errorx.DefaultErrorCode, errorx.DefaultErrorMessage)
	}
}
