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

func ListAdminsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AdminUserListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := adminuser.NewListAdminsLogic(r.Context(), svcCtx)
		resp, err := l.ListAdmins(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
