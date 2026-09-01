package adminservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type AuthenticateAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	adminUserRepo repositories.AdminUserRepository
}

func NewAuthenticateAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AuthenticateAdminLogic {
	return &AuthenticateAdminLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		adminUserRepo: repositories.NewAdminUserRepository(ctx, svcCtx.MysqlDb),
	}
}

// AuthenticateAdmin 校验管理员账号密码 通过返回身份 由 admin-api 建会话
func (l *AuthenticateAdminLogic) AuthenticateAdmin(in *admin.AuthenticateAdminReq) (*admin.AuthenticateAdminRes, error) {
	if in == nil || in.GetUsername() == "" || in.GetPassword() == "" {
		return nil, errorx.NewMsg("参数错误")
	}

	row, err := l.adminUserRepo.GetByUsername(in.GetUsername())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员失败"))
	}
	if row == nil || !utils.CheckPassword(row.PasswordHash, in.GetPassword()) {
		return nil, errorx.NewMsg("用户名或密码错误")
	}
	if row.Status != int32(admin.AdminStatus_ADMIN_ENABLED) {
		return nil, errorx.NewMsg("账号已被禁用")
	}

	return &admin.AuthenticateAdminRes{
		AdminId:  row.ID,
		Nickname: row.Nickname,
		Status:   adminStatusValue(row.Status),
	}, nil
}
