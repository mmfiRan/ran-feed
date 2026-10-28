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

func SendIMMessageHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.IMSendMessageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := im.NewSendIMMessageLogic(r.Context(), svcCtx)
		resp, err := l.SendIMMessage(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
