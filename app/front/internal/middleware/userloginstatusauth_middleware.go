// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"net/http"
	"ran-feed/app/front/internal/common/consts"
	luautils "ran-feed/app/front/internal/common/utils/lua"
	"ran-feed/app/front/internal/config"
	"ran-feed/pkg/errorx"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const (
	redisTokenKeyPrefix = "user:session"
	redisUserKeyPrefix  = "user:session:user"

	defaultSessionTTL = 7 * 24 * time.Hour
	renewRatio        = 1.0 / 3.0

	headerAuthorization = "Authorization"

	errNeedLoginMsg = "用户未登录"
)

type UserLoginStatusAuthMiddleware struct {
	redis  *redis.Redis
	config config.Config
}

func NewUserLoginStatusAuthMiddleware(redis *redis.Redis, config config.Config) *UserLoginStatusAuthMiddleware {
	return &UserLoginStatusAuthMiddleware{
		redis:  redis,
		config: config,
	}
}

func (m *UserLoginStatusAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractToken(r)
		if !ok {
			httpx.ErrorCtx(r.Context(), w, consts.ErrUserNotLogin)
			return
		}

		sessionTTL := parseSessionTTL(m.config)
		userID, err := verifyAndRenewSession(r.Context(), m.redis, token, sessionTTL)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, consts.ErrUserNotLogin)
			return
		}

		ctx := context.WithValue(r.Context(), consts.CtxKeyUserID, userID)
		ctx = context.WithValue(ctx, consts.CtxKeyToken, token)
		next(w, r.WithContext(ctx))
	}
}

func parseSessionTTL(cfg config.Config) time.Duration {
	if cfg.SessionTTL <= 0 {
		return defaultSessionTTL
	}
	return time.Duration(cfg.SessionTTL) * time.Second
}

func extractToken(r *http.Request) (string, bool) {
	authorization := strings.TrimSpace(r.Header.Get(headerAuthorization))
	if authorization == "" {
		return "", false
	}
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		t := strings.TrimSpace(parts[1])
		if t != "" {
			return t, true
		}
		return "", false
	}
	// 兼容直接传 token（无 Bearer 前缀）
	if len(parts) == 1 && parts[0] != "" {
		return parts[0], true
	}
	return "", false
}

func verifyAndRenewSession(ctx context.Context, rds *redis.Redis, token string, ttl time.Duration) (int64, error) {
	ttlSeconds := int(ttl.Seconds())
	if ttlSeconds <= 0 {
		return 0, errorx.NewMsg(errNeedLoginMsg)
	}
	threshold := int(float64(ttlSeconds) * renewRatio)
	if threshold <= 0 {
		threshold = 1
	}

	// 与 user-rpc 登录态保存逻辑保持一致：
	// tokenKey: user:session:{token} -> userId
	// userKey:  user:session:user:{userId} -> token
	tokenKey := redisTokenKeyPrefix + ":" + token
	resp, err := rds.EvalCtx(ctx, luautils.VerifyAndRenewSessionScript, []string{tokenKey}, token, redisUserKeyPrefix, ttlSeconds, threshold)
	if err != nil {
		return 0, err
	}

	var userIDStr string
	switch v := resp.(type) {
	case string:
		userIDStr = v
	case []byte:
		userIDStr = string(v)
	default:
		userIDStr = ""
	}
	userIDStr = strings.TrimSpace(userIDStr)
	if userIDStr == "" {
		return 0, errorx.NewMsg(errNeedLoginMsg)
	}
	uid, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil || uid <= 0 {
		return 0, errorx.NewMsg(errNeedLoginMsg)
	}
	return uid, nil
}
