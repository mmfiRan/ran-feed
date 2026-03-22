// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"net/http"
	"ran-feed/app/front/internal/config"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type OptionalLoginMiddleware struct {
	redis  *redis.Redis
	config config.Config
}

func NewOptionalLoginMiddleware(redis *redis.Redis, config config.Config) *OptionalLoginMiddleware {
	return &OptionalLoginMiddleware{
		redis:  redis,
		config: config,
	}
}

func (m *OptionalLoginMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractToken(r)
		if !ok {
			next(w, r)
			return
		}

		sessionTTL := parseSessionTTL(m.config)
		userID, err := verifyAndRenewSession(r.Context(), m.redis, token, sessionTTL)
		if err != nil {
			next(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUserID, userID)
		ctx = context.WithValue(ctx, ctxKeyToken, token)
		next(w, r.WithContext(ctx))
	}
}
