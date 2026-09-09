package adminauthservicelogic

import (
	"context"
	adminenums "ran-feed/app/rpc/admin/internal/common/enums"
	"ran-feed/pkg/enums"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	adminUserRepo repositories.AdminUserRepository
}

func NewGetAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAdminLogic {
	return &GetAdminLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		adminUserRepo: repositories.NewAdminUserRepository(ctx, svcCtx.MysqlDb),
	}
}

// GetAdmin 取管理员基本信息
func (l *GetAdminLogic) GetAdmin(in *admin.GetAdminReq) (*admin.GetAdminRes, error) {

	row, err := l.adminUserRepo.GetByID(in.GetAdminId())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员失败"))
	}
	if row == nil {
		return nil, errorx.NewMsg("管理员不存在")
	}

	return &admin.GetAdminRes{
		AdminId:  row.ID,
		Username: row.Username,
		Nickname: row.Nickname,
		Status:   enums.ToCommonPB(adminenums.AdminStatusEnum(row.Status)),
	}, nil
}
