package utils

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"ran-feed/pkg/consts"
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

// GetContextUserId 仅从 context 中获取 int64 类型的 C 端用户ID
func GetContextUserId(ctx context.Context) (int64, error) {
	return GetContextID(ctx, consts.CtxKeyUserID)
}

// GetContextAdminId 仅从 context 中获取 int64 类型的后台管理员ID
func GetContextAdminId(ctx context.Context) (int64, error) {
	return GetContextID(ctx, consts.CtxKeyAdminID)
}

// GetContextAdminIdWithDefault 从 context 中获取后台管理员ID
func GetContextAdminIdWithDefault(ctx context.Context) int64 {
	id, _ := GetContextAdminId(ctx)
	return id
}

// GetContextID 从 context 中按 key 取 int64 类型的 ID
func GetContextID(ctx context.Context, key string) (int64, error) {
	if ctx == nil {
		return 0, errors.New("上下文ctx为空")
	}
	v := ctx.Value(key)
	if v == nil {
		return 0, fmt.Errorf("%s不存在于上下文ctx中", key)
	}
	id, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("%s类型不是int64，实际类型为%T", key, v)
	}
	return id, nil
}
