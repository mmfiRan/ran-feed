package xxljob

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
)

type Logger interface {
	Infof(ctx context.Context, format string, args ...interface{})
	Errorf(ctx context.Context, format string, args ...interface{})
}

type goZeroLogger struct{}

func (goZeroLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	logx.WithContext(ctx).Infof(format, args...)
}

func (goZeroLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	logx.WithContext(ctx).Errorf(format, args...)
}

func logInfo(ctx context.Context, l Logger, format string, args ...interface{}) {
	if l == nil {
		goZeroLogger{}.Infof(ctx, format, args...)
		return
	}
	l.Infof(ctx, format, args...)
}

func logError(ctx context.Context, l Logger, format string, args ...interface{}) {
	if l == nil {
		goZeroLogger{}.Errorf(ctx, format, args...)
		return
	}
	l.Errorf(ctx, format, args...)
}
