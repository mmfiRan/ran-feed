// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package content

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"ran-feed/app/front/internal/logic/content"
	"ran-feed/app/front/internal/svc"
	"ran-feed/app/front/internal/types"
)

func PublishArticleHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PublishArticleReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := content.NewPublishArticleLogic(r.Context(), svcCtx)
		resp, err := l.PublishArticle(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
