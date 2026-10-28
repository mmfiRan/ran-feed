// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package im

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"ran-feed/app/front/internal/logic/im"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
)

func RevokeIMMessageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IMRevokeMessageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := im.NewRevokeIMMessageLogic(r.Context(), svcCtx)
		resp, err := l.RevokeIMMessage(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
