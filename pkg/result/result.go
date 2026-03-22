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
		logx.WithContext(ctx).Errorf("business error: %s", err.Error())
		return http.StatusOK, NewErrorResult(ce.Code, ce.Message)
	case errors.As(err, &cv):
		logx.WithContext(ctx).Errorf("validation error: %s", err.Error())
		return http.StatusOK, NewErrorResult(errorx.DefaultErrorCode, cv.Error())
	default:
		logx.WithContext(ctx).Errorf("unknown error: %s", err.Error())
		return http.StatusOK, NewErrorResult(errorx.DefaultErrorCode, errorx.DefaultErrorMessage)
	}
}
