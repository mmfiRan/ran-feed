package ratelimit

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/redis"

	rediskey "ran-feed/app/rpc/user/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/user/internal/common/utils/lua"
	"ran-feed/app/rpc/user/internal/config"
)

// LoginRateLimitResult Check 返回结果。Allowed 为 false 时 RetryAfter 给出最早窗口出清剩余秒数。
type LoginRateLimitResult struct {
	Allowed    bool
	RetryAfter int64
}

// Enabled 配置零值或负值视为关闭限频。
func Enabled(cfg config.LoginRateLimitConfig) bool {
	return cfg.WindowSeconds > 0 && cfg.MaxAttempts > 0
}

// CheckLogin 按手机号检查滑动窗口是否已达上限。
func CheckLogin(ctx context.Context, r *redis.Redis, cfg config.LoginRateLimitConfig, mobile string) (LoginRateLimitResult, error) {
	if !Enabled(cfg) || mobile == "" {
		return LoginRateLimitResult{Allowed: true}, nil
	}
	key := rediskey.BuildUserLoginFailKey(mobile)
	now := time.Now().Unix()
	res, err := r.EvalCtx(
		ctx,
		luautils.CheckLoginRateLimitScript,
		[]string{key},
		strconv.FormatInt(now, 10),
		strconv.FormatInt(cfg.WindowSeconds, 10),
		strconv.FormatInt(cfg.MaxAttempts, 10),
	)
	if err != nil {
		return LoginRateLimitResult{}, err
	}
	retryAfter, _ := res.(int64)
	if retryAfter <= 0 {
		return LoginRateLimitResult{Allowed: true}, nil
	}
	return LoginRateLimitResult{Allowed: false, RetryAfter: retryAfter}, nil
}

// RecordLoginFailure 记录一次失败，返回写入后当前窗口失败次数。
func RecordLoginFailure(ctx context.Context, r *redis.Redis, cfg config.LoginRateLimitConfig, mobile string) (int64, error) {
	if !Enabled(cfg) || mobile == "" {
		return 0, nil
	}
	key := rediskey.BuildUserLoginFailKey(mobile)
	now := time.Now().Unix()
	res, err := r.EvalCtx(
		ctx,
		luautils.RecordLoginFailureScript,
		[]string{key},
		strconv.FormatInt(now, 10),
		strconv.FormatInt(cfg.WindowSeconds, 10),
		uuid.NewString(),
	)
	if err != nil {
		return 0, err
	}
	count, _ := res.(int64)
	return count, nil
}

// ClearLoginFailures 登录成功后清零计数。
func ClearLoginFailures(ctx context.Context, r *redis.Redis, cfg config.LoginRateLimitConfig, mobile string) error {
	if !Enabled(cfg) || mobile == "" {
		return nil
	}
	_, err := r.DelCtx(ctx, rediskey.BuildUserLoginFailKey(mobile))
	return err
}