package adminauthservicelogic

import (
	"context"
	"ran-feed/app/rpc/admin/internal/common/consts"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/utils/session"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	adminUserRepo  repositories.AdminUserRepository
	permissionRepo repositories.AdminPermissionRepository
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:            ctx,
		svcCtx:         svcCtx,
		Logger:         logx.WithContext(ctx),
		adminUserRepo:  repositories.NewAdminUserRepository(ctx, svcCtx.MysqlDb),
		permissionRepo: repositories.NewAdminPermissionRepository(ctx, svcCtx.MysqlDb),
	}
}

// Login 管理员登陆
func (l *LoginLogic) Login(in *admin.LoginReq) (*admin.LoginRes, error) {
	row, err := l.adminUserRepo.GetByUsername(in.GetUsername())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员失败"))
	}
	if row == nil || !utils.CheckPassword(row.PasswordHash, in.GetPassword()) {
		return nil, errorx.NewMsg("用户名或密码错误")
	}
	if row.Status != int32(admin.AdminStatus_ADMIN_STATUS_ENABLED) {
		return nil, errorx.NewMsg("账号已被禁用")
	}

	token := uuid.NewString()
	ttlSeconds := l.sessionTTLSeconds()
	if err = session.SaveSession(l.ctx, l.svcCtx.Redis, row.ID, token, ttlSeconds); err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("保存登录态失败"))
	}

	codes, pErr := l.permissionRepo.ListCodesByAdminID(row.ID)
	if pErr != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询权限失败"))
	}

	return &admin.LoginRes{
		AdminId:     row.ID,
		Nickname:    row.Nickname,
		Token:       token,
		TtlSeconds:  int64(ttlSeconds),
		Permissions: codes,
	}, nil
}

func (l *LoginLogic) sessionTTLSeconds() int {
	if l.svcCtx.Config.SessionTTL > 0 {
		return int(l.svcCtx.Config.SessionTTL)
	}
	return consts.DefaultSessionTTLSeconds
}
