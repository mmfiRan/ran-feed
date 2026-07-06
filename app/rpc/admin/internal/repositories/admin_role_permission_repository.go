package repositories

import (
	"context"

	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminRolePermissionRepository interface {
	ListPermissionIDsByRoleIDs(roleIDs []int64) ([]int64, error)
}

type adminRolePermissionRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewAdminRolePermissionRepository(ctx context.Context, db *orm.DB) AdminRolePermissionRepository {
	return &adminRolePermissionRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// ListPermissionIDsByRoleIDs 取角色集合绑定的权限点ID集合
func (r *adminRolePermissionRepositoryImpl) ListPermissionIDsByRoleIDs(roleIDs []int64) ([]int64, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	q := query.Q
	rows, err := q.RanFeedAdminRolePermission.WithContext(r.ctx).
		Select(q.RanFeedAdminRolePermission.PermissionID).
		Where(q.RanFeedAdminRolePermission.RoleID.In(roleIDs...)).
		Where(q.RanFeedAdminRolePermission.IsDeleted.Eq(0)).
		Find()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.PermissionID > 0 {
			ids = append(ids, row.PermissionID)
		}
	}
	return ids, nil
}
