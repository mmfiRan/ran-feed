package middleware

import (
	"context"
	"net/http"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/common/rbac"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/client/adminservice"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type AdminRbacMiddleware struct {
	redis    *redis.Redis
	adminRpc adminservice.AdminService
	permTTL  int
}

func NewAdminRbacMiddleware(r *redis.Redis, adminRpc adminservice.AdminService, permTTL int) *AdminRbacMiddleware {
	if permTTL <= 0 {
		permTTL = consts.RedisAdminPermExpireSeconds
	}
	return &AdminRbacMiddleware{redis: r, adminRpc: adminRpc, permTTL: permTTL}
}

func (m *AdminRbacMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		required, need := rbac.RequiredPermission(r.Method, r.URL.Path)
		if !need {
			next(w, r)
			return
		}

		adminID, _ := r.Context().Value(consts.CtxKeyAdminID).(int64)
		if adminID <= 0 {
			httpx.ErrorCtx(r.Context(), w, consts.ErrAdminNotLogin)
			return
		}

		perms, err := rbac.LoadPermissions(r.Context(), m.redis, adminID, m.permTTL, m.loadPermissions)
		if err != nil || !rbac.HasPermission(perms, required) {
			httpx.ErrorCtx(r.Context(), w, consts.ErrAdminForbidden)
			return
		}

		next(w, r)
	}
}

func (m *AdminRbacMiddleware) loadPermissions(ctx context.Context, adminID int64) ([]string, error) {
	res, err := m.adminRpc.ListAdminPermissions(ctx, &admin.ListAdminPermissionsReq{AdminId: adminID})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.GetCodes(), nil
}
