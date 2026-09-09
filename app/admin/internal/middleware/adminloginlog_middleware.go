package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"ran-feed/app/rpc/admin/admin"
	adminauditservice "ran-feed/app/rpc/admin/client/adminauditservice"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/threading"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const loginWriteTimeout = 5 * time.Second

// loginReqBody 登录请求体 只取 username 记录不落密码
type loginReqBody struct {
	Username string `json:"username"`
}

// loginResData 登录成功响应 data 取 admin_id
type loginResData struct {
	AdminId int64 `json:"admin_id"`
}

// AdminLoginLogMiddleware 记录后台登录成功与失败 挂在 /login 路由
type AdminLoginLogMiddleware struct {
	adminRpc adminauditservice.AdminAuditService
}

func NewAdminLoginLogMiddleware(adminRpc adminauditservice.AdminAuditService) *AdminLoginLogMiddleware {
	return &AdminLoginLogMiddleware{
		adminRpc: adminRpc,
	}
}

func (m *AdminLoginLogMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := readLoginUsername(r)
		if username == "" {
			next(w, r)
			return
		}

		sw := &bodyCaptureWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}
		next(sw, r)

		status, msg, adminID := m.loginOutcome(sw)
		m.write(username, adminID, status, msg, httpx.GetRemoteAddr(r), r.Header.Get("User-Agent"))
	}
}

// readLoginUsername 从请求体中读取登录用户名
func readLoginUsername(r *http.Request) string {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ""
	}
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	var req loginReqBody
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Username
}

// loginOutcome 按统一响应结构判登录成败 成功从 data 取 admin_id 失败取 message
func (m *AdminLoginLogMiddleware) loginOutcome(sw *bodyCaptureWriter) (admin.LoginStatus, string, int64) {
	var body responseBody
	if err := json.Unmarshal(sw.body.Bytes(), &body); err != nil || body.Code == 0 {
		return admin.LoginStatus_LOGIN_STATUS_UNSPECIFIED, "", 0
	}
	if body.Code == http.StatusOK {
		return admin.LoginStatus_LOGIN_STATUS_SUCCESS, "", parseAdminID(sw.body.Bytes())
	}
	return admin.LoginStatus_LOGIN_STATUS_FAIL, body.Message, 0
}

// parseAdminID 从成功响应体 data.admin_id 取登录管理员ID
func parseAdminID(raw []byte) int64 {
	var resp struct {
		Data *loginResData `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil || resp.Data == nil {
		return 0
	}
	return resp.Data.AdminId
}

// write 异步落登录日志
func (m *AdminLoginLogMiddleware) write(username string, adminID int64, status admin.LoginStatus, msg, ip, userAgent string) {
	threading.GoSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), loginWriteTimeout)
		defer cancel()
		if _, err := m.adminRpc.WriteLoginLog(ctx, &admin.WriteLoginLogReq{
			AdminId:   adminID,
			Username:  username,
			Ip:        ip,
			UserAgent: userAgent,
			Status:    status,
			Msg:       msg,
		}); err != nil {
			logx.WithContext(ctx).Errorf("写登录审计失败 username=%s err=%v", username, err)
		}
	})
}
