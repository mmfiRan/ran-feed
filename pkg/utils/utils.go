package utils

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/grpc/metadata"
)

func GetSixDigitsVerificationCode(length ...int) string {
	if len(length) == 0 {
		length = append(length, 6)
	}
	lens := length[0]
	if lens < 6 || lens > 8 {
		lens = 6
	}
	rand.New(rand.NewSource(time.Now().UnixNano()))
	code := ""
	for i := 0; i < lens; i++ {
		code += fmt.Sprintf("%d", rand.Intn(10))
	}
	return code
}

func GetContextUserIdWithDefault(ctx context.Context) int64 {
	id, _ := GetContextUserId(ctx)
	return id
}

// GetContextUserId 仅从 context 中获取 int64 类型的 "user_id"，并返回 (id, error)
// - 当 ctx 为空、未找到 user_id、或类型不是 int64 时返回错误；不再发生 panic
func GetContextUserId(ctx context.Context) (int64, error) {
	//// 默认返回一个固定的用户ID，用于测试
	//return snowflake.GenID(), nil
	if ctx == nil {
		return 0, errors.New("上下文ctx为空")
	}
	v := ctx.Value("user_id")
	if v == nil {
		return 0, errors.New("user_id不存在于上下文ctx中")
	}
	id, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("user_id类型不是int64，实际类型为%T", v)
	}
	return id, nil
}

func CombinedErrorAndMessage(err error, message string) (error, string) {
	if err == nil {
		return errors.New(message), message
	}
	err = fmt.Errorf("%s: %w", message, err)
	return err, message
}

// GetUserIDFromRpcMetadata 从RPC的metadata中获取用户id
func GetUserIDFromRpcMetadata(ctx context.Context, metadataKey ...string) (int64, error) {
	if ctx == nil {
		return 0, errors.New("上下文ctx为空")
	}

	// 设置默认的metadata键名
	key := "x-user-id"
	if len(metadataKey) > 0 && metadataKey[0] != "" {
		key = metadataKey[0]
	}

	// 从上下文获取metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, errors.New("无法从上下文获取metadata")
	}

	// 获取用户ID值
	vals := md.Get(key)
	if len(vals) == 0 {
		return 0, fmt.Errorf("metadata中未找到键: %s", key)
	}

	// 解析用户ID
	userID, err := strconv.ParseInt(vals[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析用户ID失败: %w", err)
	}

	// 验证用户ID有效性
	if userID <= 0 {
		return 0, errors.New("用户ID必须大于0")
	}

	return userID, nil
}

// GetUserIDFromRpcMetadataSafe 从RPC的metadata中安全获取用户id，不会返回错误
func GetUserIDFromRpcMetadataSafe(ctx context.Context, metadataKey ...string) int64 {
	userID, _ := GetUserIDFromRpcMetadata(ctx, metadataKey...)
	return userID
}

func GetUserIdFromHttpHeader(r *http.Request) (int64, error) {
	val := r.Header.Get("X-User-Id")
	if val == "" {
		return 0, errors.New("用户ID不存在于请求头中")
	}
	// 获取出来之后设置到请求上下文中
	userID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("解析X-User-Id失败: %w", err)
	}
	context.WithValue(r.Context(), "user_id", userID)
	return userID, nil
}
