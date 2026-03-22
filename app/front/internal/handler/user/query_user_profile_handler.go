// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package user

import (
	"net/http"

	"ran-feed/app/front/internal/logic/user"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func QueryUserProfileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.QueryUserProfileReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := user.NewQueryUserProfileLogic(r.Context(), svcCtx)
		resp, err := l.QueryUserProfile(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
