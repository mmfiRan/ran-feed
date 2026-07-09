package adminservicelogic

import (
	"context"
	"strings"

	"ran-feed/app/rpc/admin/admin"
	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/app/rpc/admin/internal/repositories"
	"ran-feed/app/rpc/admin/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
)

// adminSaltBytes 管理员密码盐字节数
const adminSaltBytes = 16

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

// CreateAdmin 建管理员 username 唯一 生成盐加哈希 事务内建账号并绑角色
func (l *CreateAdminLogic) CreateAdmin(in *admin.CreateAdminReq) (*admin.CreateAdminRes, error) {
	username := strings.TrimSpace(in.GetUsername())
	if username == "" || in.GetPassword() == "" {
		return nil, errorx.NewMsg("参数错误")
	}

	existing, err := l.adminUserRepo.GetByUsername(username)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询管理员失败"))
	}
	if existing != nil {
		return nil, errorx.NewMsg("用户名已存在")
	}

	salt, err := utils.GenerateSalt(adminSaltBytes)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("生成密码盐失败"))
	}
	hash, err := utils.HashPassword(in.GetPassword() + salt)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("生成密码哈希失败"))
	}

	roleIDs := dedupInt64(in.GetRoleIds())
	var newID int64
	err = query.Q.Transaction(func(tx *query.Query) error {
		id, e := l.adminUserRepo.WithTx(tx).Create(&model.RanFeedAdminUser{
			Username:     username,
			PasswordHash: hash,
			PasswordSalt: salt,
			Nickname:     in.GetNickname(),
			Status:       int32(admin.AdminStatus_ADMIN_ENABLED),
			CreatedBy:    in.GetOperatorId(),
			UpdatedBy:    in.GetOperatorId(),
		})
		if e != nil {
			return e
		}
		newID = id
		return l.userRoleRepo.WithTx(tx).BatchCreate(buildUserRoleRows(id, roleIDs, in.GetOperatorId()))
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建管理员失败"))
	}
	return &admin.CreateAdminRes{Id: newID}, nil
}

// buildUserRoleRows 组装管理员角色绑定行 跳过非正 roleID
func buildUserRoleRows(adminID int64, roleIDs []int64, operatorID int64) []*model.RanFeedAdminUserRole {
	rows := make([]*model.RanFeedAdminUserRole, 0, len(roleIDs))
	for _, rid := range roleIDs {
		if rid <= 0 {
			continue
		}
		rows = append(rows, &model.RanFeedAdminUserRole{
			AdminUserID: adminID,
			RoleID:      rid,
			CreatedBy:   operatorID,
			UpdatedBy:   operatorID,
		})
	}
	return rows
}
