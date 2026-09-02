package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/utils/session"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResetAdminPasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	adminUserRepo repositories.AdminUserRepository
}

func NewResetAdminPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetAdminPasswordLogic {
	return &ResetAdminPasswordLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		adminUserRepo: repositories.NewAdminUserRepository(ctx, svcCtx.MysqlDb),
	}
}

// ResetAdminPassword 重置管理员密码 bcrypt 重新哈希
func (l *ResetAdminPasswordLogic) ResetAdminPassword(in *admin.ResetAdminPasswordReq) (*admin.ResetAdminPasswordRes, error) {

	hash, err := utils.HashPassword(in.GetNewPassword())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("生成密码哈希失败"))
	}

	affected, err := l.adminUserRepo.UpdatePassword(in.GetId(), hash, in.GetOperatorId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("重置密码失败"))
	}
	if affected == 0 {
		return nil, errorx.NewMsg("管理员不存在")
	}

	// 重置即踢下线
	if e := session.RemoveByAdminID(l.ctx, l.svcCtx.Redis, in.GetId()); e != nil {
		l.Errorf("重置密码后踢下线失败 adminID=%d err=%v", in.GetId(), e)
	}
	return &admin.ResetAdminPasswordRes{}, nil
}
