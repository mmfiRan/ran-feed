package repositories

import (
	"context"

	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminRolePermissionRepository interface {
	WithTx(tx *query.Query) AdminRolePermissionRepository
	// ListPermissionIDsByRoleIDs 取角色集合绑定的权限点ID集合
	ListPermissionIDsByRoleIDs(roleIDs []int64) ([]int64, error)
	// DeleteByRoleID 物理删角色的全部权限点绑定 返回影响行数
	DeleteByRoleID(roleID int64) (int64, error)
	// BatchCreate 批量建绑定
	BatchCreate(rows []*model.RanFeedAdminRolePermission) error
}

type adminRolePermissionRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewAdminRolePermissionRepository(ctx context.Context, db *orm.DB) AdminRolePermissionRepository {
	return &adminRolePermissionRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *adminRolePermissionRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *adminRolePermissionRepositoryImpl) WithTx(tx *query.Query) AdminRolePermissionRepository {
	return &adminRolePermissionRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

// ListPermissionIDsByRoleIDs 取角色集合绑定的权限点ID集合
func (r *adminRolePermissionRepositoryImpl) ListPermissionIDsByRoleIDs(roleIDs []int64) ([]int64, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	q := r.getQuery().RanFeedAdminRolePermission
	rows, err := q.WithContext(r.ctx).
		Select(q.PermissionID).
		Where(q.RoleID.In(roleIDs...)).
		Where(q.IsDeleted.Eq(0)).
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

// DeleteByRoleID 物理删角色的全部权限点绑定 返回影响行数
func (r *adminRolePermissionRepositoryImpl) DeleteByRoleID(roleID int64) (int64, error) {
	q := r.getQuery().RanFeedAdminRolePermission
	res, err := q.WithContext(r.ctx).Where(q.RoleID.Eq(roleID)).Delete()
	if err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}

// BatchCreate 批量建绑定
func (r *adminRolePermissionRepositoryImpl) BatchCreate(rows []*model.RanFeedAdminRolePermission) error {
	if len(rows) == 0 {
		return nil
	}
	return r.getQuery().RanFeedAdminRolePermission.WithContext(r.ctx).Create(rows...)
}
