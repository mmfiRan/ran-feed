package middleware

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/config"
	pkgconsts "ran-feed/pkg/consts"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const defaultSessionTTL = 7 * 24 * time.Hour

type AdminAuthMiddleware struct {
	redis  *redis.Redis
	config config.Config
}

func NewAdminAuthMiddleware(r *redis.Redis, c config.Config) *AdminAuthMiddleware {
	return &AdminAuthMiddleware{
		redis:  r,
		config: c,
	}
}

func (m *AdminAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractToken(r)
		if !ok {
			httpx.ErrorCtx(r.Context(), w, consts.ErrAdminNotLogin)
			return
		}
		adminID, err := m.verifyAndRenew(r.Context(), token)
		if err != nil || adminID <= 0 {
			httpx.ErrorCtx(r.Context(), w, consts.ErrAdminNotLogin)
			return
		}
		ctx := context.WithValue(r.Context(), pkgconsts.CtxKeyAdminID, adminID)
		ctx = context.WithValue(ctx, pkgconsts.CtxKeyToken, token)
		next(w, r.WithContext(ctx))
	}
}

func (m *AdminAuthMiddleware) sessionTTL() time.Duration {
	if m.config.SessionTTL <= 0 {
		return defaultSessionTTL
	}
	return time.Duration(m.config.SessionTTL) * time.Second
}

func (m *AdminAuthMiddleware) verifyAndRenew(ctx context.Context, token string) (int64, error) {
	val, err := m.redis.GetCtx(ctx, consts.BuildAdminSessionKey(token))
	if err != nil {
		return 0, err
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return 0, nil
	}
	adminID, parseErr := strconv.ParseInt(val, 10, 64)
	if parseErr != nil {
		// 会话值损坏 记录日志并视为未登录
		logx.WithContext(ctx).Errorf("管理员会话值损坏 key=%s err=%v", consts.BuildAdminSessionKey(token), parseErr)
		return 0, nil
	}
	if adminID <= 0 {
		return 0, nil
	}

	ttl := time.Duration(m.sessionTTL().Seconds()) * time.Second

	// 滑动续期
	if err = m.redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		pipe.Expire(ctx, consts.BuildAdminSessionKey(token), ttl)
		pipe.Expire(ctx, consts.BuildAdminSessionUIDKey(adminID), ttl)
		return nil
	}); err != nil {
		return 0, err
	}
	return adminID, nil
}

func extractToken(r *http.Request) (string, bool) {
	authorization := strings.TrimSpace(r.Header.Get(consts.HeaderAuthorization))
	if authorization == "" {
		return "", false
	}
	parts := strings.SplitN(authorization, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		t := strings.TrimSpace(parts[1])
		return t, t != ""
	}
	if len(parts) == 1 && parts[0] != "" {
		return parts[0], true
	}
	return "", false
}
