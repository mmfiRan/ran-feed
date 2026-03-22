package errorx

import (
	"bytes"
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"runtime"
)

const (
	DefaultErrorCode    uint32 = 500
	DefaultErrorMessage string = "服务内部错误"
)

type BizError struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

func New(message string, code uint32) *BizError {
	return &BizError{
		Code:    code,
		Message: message,
	}
}

func NewMsg(message string) *BizError {
	return New(message, DefaultErrorCode)
}

func (e *BizError) Error() string {
	return e.Message
}

// Wrap 只打印日志和堆栈
func Wrap(ctx context.Context, sysErr error, bizErr *BizError) *BizError {
	if sysErr == nil {
		return bizErr
	}

	stack := captureStack(2)
	logx.WithContext(ctx).WithCallerSkip(1).Errorf(
		"%s: %v\nStack:\n%s",
		bizErr.Message,
		sysErr,
		formatStack(stack),
	)

	return bizErr
}

func captureStack(skip int) []uintptr {
	const depth = 32
	pcs := make([]uintptr, depth)
	n := runtime.Callers(skip, pcs)
	return pcs[:n]
}

func formatStack(pcs []uintptr) string {
	var buf bytes.Buffer
	frames := runtime.CallersFrames(pcs)
	for {
		f, more := frames.Next()
		buf.WriteString(fmt.Sprintf("%s:%d %s\n", f.File, f.Line, f.Function))
		if !more {
			break
		}
	}
	return buf.String()
}
