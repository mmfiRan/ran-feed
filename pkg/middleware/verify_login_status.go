package middleware

import (
	"context"
	"encoding/json"
	"github.com/zeromicro/go-zero/core/logx"
	"net/http"
	"ran-feed/pkg/result"
)

func VerifyLoginStatus(w http.ResponseWriter, r *http.Request) (*http.Request, bool) {
	userId := getUserIdFromCtx(r.Context())
	if userId == 0 {
		body, _ := json.Marshal(result.NewErrorResult(http.StatusOK, "用户未登录"))
		w.Write(body)
		return r, false
	}

	// 您可以在这里添加额外的逻辑，例如去 Redis 校验 Token 是否有效或已过期

	// 将统一处理过的int64类型的userId放回context中，方便后续的handler使用
	ctx := context.WithValue(r.Context(), "userId", userId)
	return r.WithContext(ctx), true
}

func getUserIdFromCtx(ctx context.Context) int64 {
	var userId int64
	val := ctx.Value("userId")
	if val == nil {
		return 0
	}

	switch v := val.(type) {
	case json.Number:
		userId, _ = v.Int64()
	case float64:
		userId = int64(v)
	case int64:
		userId = v
	}
	return userId
}

func BuildAuthFailHandler(w http.ResponseWriter, r *http.Request, err error) {
	logx.Errorf("登录鉴权失败，err: %v", err)
	body, _ := json.Marshal(result.NewErrorResult(http.StatusOK, "用户未登录"))
	w.Write(body)
}
