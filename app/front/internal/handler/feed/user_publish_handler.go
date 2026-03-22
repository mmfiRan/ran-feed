// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package feed

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"ran-feed/app/front/internal/logic/feed"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
)

func UserPublishHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UserPublishFeedReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := feed.NewUserPublishLogic(r.Context(), svcCtx)
		resp, err := l.UserPublish(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
