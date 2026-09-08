// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package audit

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"ran-feed/app/admin/internal/logic/audit"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
)

func ListLoginLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AdminLoginLogListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := audit.NewListLoginLogsLogic(r.Context(), svcCtx)
		resp, err := l.ListLoginLogs(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
