package middleware

import (
	"context"
	"net/http"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/common/rbac"
	"ran-feed/app/admin/internal/docmeta"
	"ran-feed/app/rpc/admin/admin"
	adminauthservice "ran-feed/app/rpc/admin/client/adminauthservice"
	pkgconsts "ran-feed/pkg/consts"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/httpx"
)

type AdminRbacMiddleware struct {
	redis    *redis.Redis
	adminRpc adminauthservice.AdminAuthService
	permTTL  int
}

func NewAdminRbacMiddleware(r *redis.Redis, adminRpc adminauthservice.AdminAuthService, permTTL int) *AdminRbacMiddleware {
	if permTTL <= 0 {
		permTTL = consts.RedisAdminPermExpireSeconds
	}
	return &AdminRbacMiddleware{
		redis:    r,
		adminRpc: adminRpc,
		permTTL:  permTTL,
	}
}

func (m *AdminRbacMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		adminID, _ := r.Context().Value(pkgconsts.CtxKeyAdminID).(int64)
		if adminID <= 0 {
			httpx.ErrorCtx(r.Context(), w, consts.ErrAdminNotLogin)
			return
		}

		// 所需权限点由 docmeta.Inject 从路由 @doc 注入 ctx
		required, ok := docmeta.Value(r.Context(), "permission")
		if !ok || required == "" {
			logx.WithContext(r.Context()).Errorf("路由缺少 permission 声明 需在 api 文件补 @doc 并重跑 rbacgen path=%s", r.URL.Path)
			httpx.ErrorCtx(r.Context(), w, consts.ErrAdminForbidden)
			return
		}

		perms, err := rbac.LoadPermissions(r.Context(), m.redis, adminID, m.permTTL, m.loadPermissions)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("加载权限时出错 err=%s", err.Error())
		}
		if !rbac.HasPermission(perms, required) {
			httpx.ErrorCtx(r.Context(), w, consts.ErrAdminForbidden)
			return
		}

		next(w, r)
	}
}

func (m *AdminRbacMiddleware) loadPermissions(ctx context.Context, adminID int64) ([]string, error) {
	res, err := m.adminRpc.ListAdminPermissions(ctx, &admin.ListAdminPermissionsReq{
		AdminId: adminID,
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}
	return res.GetCodes(), nil
}
