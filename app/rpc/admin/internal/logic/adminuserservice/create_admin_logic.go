package adminuserservicelogic

import (
	"context"
	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/common/logichelper"
	adminutils "ran-feed/app/rpc/admin/internal/common/utils"
	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
	adminUserRepo repositories.AdminUserRepository
	userRoleRepo  repositories.AdminUserRoleRepository
}

func NewCreateAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAdminLogic {
	return &CreateAdminLogic{
		ctx:           ctx,
		svcCtx:        svcCtx,
		Logger:        logx.WithContext(ctx),
		adminUserRepo: repositories.NewAdminUserRepository(ctx, svcCtx.MysqlDb),
		userRoleRepo:  repositories.NewAdminUserRoleRepository(ctx, svcCtx.MysqlDb),
	}
}

// CreateAdmin 创建管理员
func (l *CreateAdminLogic) CreateAdmin(in *admin.CreateAdminReq) (*admin.CreateAdminRes, error) {
	username := in.GetUsername()
	existing, err := l.adminUserRepo.GetByUsername(username)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员失败"))
	}
	if existing != nil {
		return nil, errorx.NewMsg("用户名已存在")
	}

	hash, err := utils.HashPassword(in.GetPassword())
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("生成密码哈希失败"))
	}

	roleIDs := adminutils.Dedup[int64](in.GetRoleIds())
	adminID := snowflake.GenID()
	err = query.Q.Transaction(func(tx *query.Query) error {
		if e := l.adminUserRepo.WithTx(tx).Create(&model.RanFeedAdminUser{
			ID:           adminID,
			Username:     username,
			PasswordHash: hash,
			Nickname:     in.GetNickname(),
			Status:       int32(admin.AdminStatus_ADMIN_STATUS_ENABLED),
			CreatedBy:    in.GetOperatorId(),
			UpdatedBy:    in.GetOperatorId(),
		}); e != nil {
			return e
		}
		return l.userRoleRepo.WithTx(tx).BatchCreate(logichelper.BuildUserRoleRows(adminID, roleIDs, in.GetOperatorId()))
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建管理员失败"))
	}
	return &admin.CreateAdminRes{
		Id: adminID,
	}, nil
}
