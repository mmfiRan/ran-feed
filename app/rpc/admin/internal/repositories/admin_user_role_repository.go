package repositories

import (
	"context"

	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUserRoleRepository interface {
	ListRoleIDsByAdminID(adminID int64) ([]int64, error)
}

type adminUserRoleRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewAdminUserRoleRepository(ctx context.Context, db *orm.DB) AdminUserRoleRepository {
	return &adminUserRoleRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// ListRoleIDsByAdminID 取管理员绑定的角色ID集合
func (r *adminUserRoleRepositoryImpl) ListRoleIDsByAdminID(adminID int64) ([]int64, error) {
	q := query.Q
	rows, err := q.RanFeedAdminUserRole.WithContext(r.ctx).
		Select(q.RanFeedAdminUserRole.RoleID).
		Where(q.RanFeedAdminUserRole.AdminUserID.Eq(adminID)).
		Where(q.RanFeedAdminUserRole.IsDeleted.Eq(0)).
		Find()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.RoleID > 0 {
			ids = append(ids, row.RoleID)
		}
	}
	return ids, nil
}
