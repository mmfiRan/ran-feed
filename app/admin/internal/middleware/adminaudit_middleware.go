package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ran-feed/app/rpc/admin/admin"
	adminauditservice "ran-feed/app/rpc/admin/client/adminauditservice"
	"ran-feed/app/admin/internal/docmeta"
	pkgconsts "ran-feed/pkg/consts"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const auditWriteTimeout = 5 * time.Second

// responseBody 统一响应结构 用于读业务 code 判操作成败
type responseBody struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

type AdminAuditMiddleware struct {
	adminRpc adminauditservice.AdminAuditService
}

func NewAdminAuditMiddleware(adminRpc adminauditservice.AdminAuditService) *AdminAuditMiddleware {
	return &AdminAuditMiddleware{adminRpc: adminRpc}
}

func (m *AdminAuditMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只审计写操作 GET 不记
		if r.Method == http.MethodGet {
			next(w, r)
			return
		}

		start := time.Now()
		sw := &bodyCaptureWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next(sw, r)

		adminID, _ := r.Context().Value(pkgconsts.CtxKeyAdminID).(int64)
		if adminID <= 0 {
			return
		}

		status, errorMsg := sw.operateOutcome()
		action, title := m.routeMeta(r)

		m.write(adminID, action, title, status, errorMsg, httpx.GetRemoteAddr(r),
			r.Header.Get("User-Agent"), time.Since(start).Milliseconds())
	}
}

// routeMeta 取路由 @doc 注入的权限码与描述作 action/title 未命中回退 METHOD path
func (m *AdminAuditMiddleware) routeMeta(r *http.Request) (action, title string) {
	meta := docmeta.FromContext(r.Context())
	if p, ok := meta["permission"]; ok && p != "" {
		action = p
	} else {
		action = r.Method + " " + r.URL.Path
	}
	if d, ok := meta["description"]; ok {
		title = d
	}
	return action, title
}

// write 异步落审计
func (m *AdminAuditMiddleware) write(adminID int64, action, title string, status admin.OperateStatus,
	errorMsg, ip, userAgent string, costTime int64) {
	threading.GoSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), auditWriteTimeout)
		defer cancel()
		if _, err := m.adminRpc.WriteOperationLog(ctx, &admin.WriteOperationLogReq{
			AdminId:   adminID,
			Action:    action,
			Title:     title,
			Status:    status,
			ErrorMsg:  errorMsg,
			CostTime:  costTime,
			Ip:        ip,
			UserAgent: userAgent,
		}); err != nil {
			logx.WithContext(ctx).Errorf("写操作审计失败 adminID=%d action=%s err=%v", adminID, action, err)
		}
	})
}

// bodyCaptureWriter 捕获状态码与响应体供成败判定
type bodyCaptureWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *bodyCaptureWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *bodyCaptureWriter) Write(p []byte) (int, error) {
	w.body.Write(p)
	return w.ResponseWriter.Write(p)
}

// operateOutcome 按统一响应结构判操作成败 HTTP 非 2xx 直接失败
func (w *bodyCaptureWriter) operateOutcome() (admin.OperateStatus, string) {
	if w.status >= http.StatusBadRequest {
		return admin.OperateStatus_OPERATE_STATUS_FAIL, strconv.Itoa(w.status)
	}
	var body responseBody
	if err := json.Unmarshal(w.body.Bytes(), &body); err != nil || body.Code == 0 {
		// 非统一 JSON 结构按 HTTP 状态判
		return admin.OperateStatus_OPERATE_STATUS_SUCCESS, ""
	}
	if body.Code == http.StatusOK {
		return admin.OperateStatus_OPERATE_STATUS_SUCCESS, ""
	}
	return admin.OperateStatus_OPERATE_STATUS_FAIL, body.Message
}
