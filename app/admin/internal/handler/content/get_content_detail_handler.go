// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package content

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"ran-feed/app/admin/internal/logic/content"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
)

func GetContentDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AdminContentDetailReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := content.NewGetContentDetailLogic(r.Context(), svcCtx)
		resp, err := l.GetContentDetail(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
