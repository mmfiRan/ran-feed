package adminuserservicelogic

import (
	"context"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/logichelper"
	"ran-feed/app/rpc/admin/internal/common/utils/session"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type SetAdminStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	adminUserRepo repositories.AdminUserRepository
}

func NewSetAdminStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAdminStatusLogic {
	return &SetAdminStatusLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		adminUserRepo: repositories.NewAdminUserRepository(ctx, svcCtx.MysqlDb),
	}
}

// SetAdminStatus 启用禁用管理员
func (l *SetAdminStatusLogic) SetAdminStatus(in *admin.SetAdminStatusReq) (*emptypb.Empty, error) {
	if in.GetId() <= 0 {
		return nil, errorx.NewMsg("管理员ID错误")
	}
	status := in.GetStatus()
	if status != admin.AdminStatus_ADMIN_STATUS_ENABLED && status != admin.AdminStatus_ADMIN_STATUS_DISABLED {
		return nil, errorx.NewMsg("不支持的状态")
	}
	if logichelper.IsSelfDisable(in.GetId(), in.GetOperatorId(), status) {
		return nil, errorx.NewMsg("不能禁用自己")
	}

	affected, err := l.adminUserRepo.UpdateStatus(in.GetId(), int32(status), in.GetOperatorId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("更新管理员状态失败"))
	}
	if affected == 0 {
		return nil, errorx.NewMsg("管理员不存在")
	}

	// 禁用即踢下线
	if status == admin.AdminStatus_ADMIN_STATUS_DISABLED {
		if e := session.RemoveByAdminID(l.ctx, l.svcCtx.Redis, in.GetId()); e != nil {
			l.Errorf("禁用后踢下线失败 adminID=%d err=%v", in.GetId(), e)
		}
	}
	return &emptypb.Empty{}, nil
}
