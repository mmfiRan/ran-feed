// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package user

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"ran-feed/app/admin/internal/logic/user"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
)

func ListUsersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CUserListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := user.NewListUsersLogic(r.Context(), svcCtx)
		resp, err := l.ListUsers(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
