// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package adminuser

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"ran-feed/app/admin/internal/logic/adminuser"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
)

func GetAdminDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AdminUserDetailReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := adminuser.NewGetAdminDetailLogic(r.Context(), svcCtx)
		resp, err := l.GetAdminDetail(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
