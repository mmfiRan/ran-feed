// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package auth

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/admin/internal/common/consts"
	"ran-feed/app/admin/internal/svc"
	"ran-feed/app/admin/internal/types"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/pkg/errorx"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.AdminLoginReq) (resp *types.AdminLoginRes, err error) {
	if req == nil || req.Username == nil || req.Password == nil {
		return nil, errorx.NewMsg("参数错误")
	}

	authRes, err := l.svcCtx.AdminRpc.AuthenticateAdmin(l.ctx, &admin.AuthenticateAdminReq{
		Username: *req.Username,
		Password: *req.Password,
	})
	if err != nil {
		return nil, err
	}
	if authRes == nil || authRes.GetAdminId() <= 0 {
		return nil, errorx.NewMsg("用户名或密码错误")
	}

	token := uuid.NewString()
	ttlSeconds := l.sessionTTLSeconds()
	if err = l.saveSession(authRes.GetAdminId(), token, ttlSeconds); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存登录态失败"))
	}

	permissions := l.loadPermissions(authRes.GetAdminId())

	return &types.AdminLoginRes{
		AdminId:     authRes.GetAdminId(),
		Token:       token,
		ExpiredAt:   time.Now().Add(time.Duration(ttlSeconds) * time.Second).Unix(),
		Nickname:    authRes.GetNickname(),
		Permissions: permissions,
	}, nil
}

func (l *LoginLogic) sessionTTLSeconds() int {
	if l.svcCtx.Config.SessionTTL > 0 {
		return int(l.svcCtx.Config.SessionTTL)
	}
	return consts.RedisAdminSessionExpireSecondsDefault
}

// saveSession 写双向会话 顶掉旧 token 防重复登录遗留
func (l *LoginLogic) saveSession(adminID int64, token string, ttlSeconds int) error {
	adminKey := consts.BuildAdminSessionUIDKey(adminID)
	if oldToken, gErr := l.svcCtx.Redis.GetCtx(l.ctx, adminKey); gErr == nil && oldToken != "" {
		_, _ = l.svcCtx.Redis.DelCtx(l.ctx, consts.BuildAdminSessionKey(oldToken))
	}
	if err := l.svcCtx.Redis.SetexCtx(l.ctx, consts.BuildAdminSessionKey(token), strconv.FormatInt(adminID, 10), ttlSeconds); err != nil {
		return err
	}
	return l.svcCtx.Redis.SetexCtx(l.ctx, adminKey, token, ttlSeconds)
}

// loadPermissions 权限点集合供前端渲染菜单 失败降级空不阻断登录
func (l *LoginLogic) loadPermissions(adminID int64) []string {
	permRes, err := l.svcCtx.AdminRpc.ListAdminPermissions(l.ctx, &admin.ListAdminPermissionsReq{AdminId: adminID})
	if err != nil {
		logx.WithContext(l.ctx).Errorf("查询管理员权限失败 adminID=%d err=%v", adminID, err)
		return nil
	}
	if permRes == nil {
		return nil
	}
	return permRes.GetCodes()
}
