package interceptor

import (
	"context"
	"errors"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/grpcx"
	"runtime/debug"

	"github.com/zeromicro/go-zero/core/logc"
	"google.golang.org/grpc"
)

// ClientGrpcInterceptor Client 端拦截器
func ClientGrpcInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// 调用远程方法
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			return grpcx.ToError(ctx, err)
		}
		return nil
	}
}

// ServerGrpcInterceptor Server 端拦截器
func ServerGrpcInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		// Panic 恢复
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				logc.Errorf(ctx, "[gRPC Server Panic] method=%s, panic=%v\n%s", info.FullMethod, r, stack)

				// 返回兜底错误
				bizErr := errorx.New(errorx.DefaultErrorMessage, errorx.DefaultErrorCode)
				err = grpcx.FromError(bizErr)
			}
		}()

		// 执行实际的 handler
		resp, err = handler(ctx, req)

		if err != nil {
			// 记录非业务错误的日志
			var bizErr *errorx.BizError
			if !errors.As(err, &bizErr) {
				logc.Errorf(ctx, "[gRPC Server BizError] method=%s, error=%v", info.FullMethod, bizErr)
			}

			// 转换错误（核心调用）
			return resp, grpcx.FromError(err)
		}

		return resp, nil
	}
}
