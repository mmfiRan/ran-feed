package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/client/adminservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const auditWriteTimeout = 5 * time.Second

type AdminAuditMiddleware struct {
	adminRpc adminservice.AdminService
}

func NewAdminAuditMiddleware(adminRpc adminservice.AdminService) *AdminAuditMiddleware {
	return &AdminAuditMiddleware{adminRpc: adminRpc}
}

func (m *AdminAuditMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只审计写操作 GET 不记
		if r.Method == http.MethodGet {
			next(w, r)
			return
		}

		sw := &statusWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next(sw, r)

		adminID, _ := r.Context().Value(consts.CtxKeyAdminID).(int64)
		if adminID <= 0 {
			return
		}
		m.write(adminID, r.Method+" "+r.URL.Path, httpx.GetRemoteAddr(r), strconv.Itoa(sw.status))
	}
}

// write 异步落审计
func (m *AdminAuditMiddleware) write(adminID int64, action, ip, result string) {
	threading.GoSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), auditWriteTimeout)
		defer cancel()
		if _, err := m.adminRpc.WriteOperationLog(ctx, &admin.WriteOperationLogReq{
			AdminId: adminID,
			Action:  action,
			Ip:      ip,
			Result:  result,
		}); err != nil {
			logx.WithContext(ctx).Errorf("写操作审计失败 adminID=%d action=%s err=%v", adminID, action, err)
		}
	})
}

// statusWriter 捕获响应状态码供审计记录
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
