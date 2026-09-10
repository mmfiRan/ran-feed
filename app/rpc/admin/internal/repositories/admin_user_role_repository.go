package repositories

import (
	"context"
	"ran-feed/pkg/enums"

	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUserRoleRepository interface {
	WithTx(tx *query.Query) AdminUserRoleRepository
	// ListRoleIDsByAdminID 取管理员绑定的角色ID集合
	ListRoleIDsByAdminID(adminID int64) ([]int64, error)
	// ListRoleIDsByAdminIDs 取管理员集合各自绑定的角色ID
	ListRoleIDsByAdminIDs(adminIDs []int64) (map[int64][]int64, error)
	// ListAdminIDsByRoleID 取绑定该角色的管理员ID集合
	ListAdminIDsByRoleID(roleID int64) ([]int64, error)
	// DeleteByRoleID 物理删该角色的全部管理员绑定
	DeleteByRoleID(roleID int64) (int64, error)
	// DeleteByAdminID 物理删该管理员的全部角色绑定 返回影响行数
	DeleteByAdminID(adminID int64) (int64, error)
	// BatchCreate 批量建绑定
	BatchCreate(rows []*model.RanFeedAdminUserRole) error
}

type adminUserRoleRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewAdminUserRoleRepository(ctx context.Context, db *orm.DB) AdminUserRoleRepository {
	return &adminUserRoleRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *adminUserRoleRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *adminUserRoleRepositoryImpl) WithTx(tx *query.Query) AdminUserRoleRepository {
	return &adminUserRoleRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

// ListRoleIDsByAdminID 取管理员绑定的角色ID集合
func (r *adminUserRoleRepositoryImpl) ListRoleIDsByAdminID(adminID int64) ([]int64, error) {
	q := r.getQuery().RanFeedAdminUserRole
	rows, err := q.WithContext(r.ctx).
		Select(q.RoleID).
		Where(q.AdminUserID.Eq(adminID)).
		Where(q.IsDeleted.Eq(0)).
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

// ListRoleIDsByAdminIDs 取管理员集合各自绑定的角色ID
func (r *adminUserRoleRepositoryImpl) ListRoleIDsByAdminIDs(adminIDs []int64) (map[int64][]int64, error) {
	if len(adminIDs) == 0 {
		return map[int64][]int64{}, nil
	}
	q := r.getQuery().RanFeedAdminUserRole
	rows, err := q.WithContext(r.ctx).
		Select(q.AdminUserID, q.RoleID).
		Where(q.AdminUserID.In(adminIDs...)).
		Where(q.IsDeleted.Eq(0)).
		Find()
	if err != nil {
		return nil, err
	}
	out := make(map[int64][]int64, len(adminIDs))
	for _, row := range rows {
		if row == nil || row.RoleID <= 0 {
			continue
		}
		out[row.AdminUserID] = append(out[row.AdminUserID], row.RoleID)
	}
	return out, nil
}

// DeleteByAdminID 物理删该管理员的全部角色绑定 返回影响行数
func (r *adminUserRoleRepositoryImpl) DeleteByAdminID(adminID int64) (int64, error) {
	q := r.getQuery().RanFeedAdminUserRole
	res, err := q.WithContext(r.ctx).Where(q.AdminUserID.Eq(adminID)).Delete()
	if err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}

// ListAdminIDsByRoleID 取绑定该角色的管理员ID集合
func (r *adminUserRoleRepositoryImpl) ListAdminIDsByRoleID(roleID int64) ([]int64, error) {
	q := r.getQuery().RanFeedAdminUserRole
	rows, err := q.WithContext(r.ctx).
		Select(q.AdminUserID).
		Where(q.RoleID.Eq(roleID)).
		Where(q.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Find()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.AdminUserID > 0 {
			ids = append(ids, row.AdminUserID)
		}
	}
	return ids, nil
}

// DeleteByRoleID 物理删该角色的全部管理员绑定 返回影响行数
func (r *adminUserRoleRepositoryImpl) DeleteByRoleID(roleID int64) (int64, error) {
	q := r.getQuery().RanFeedAdminUserRole
	res, err := q.WithContext(r.ctx).Where(q.RoleID.Eq(roleID)).Delete()
	if err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}

// BatchCreate 批量建绑定
func (r *adminUserRoleRepositoryImpl) BatchCreate(rows []*model.RanFeedAdminUserRole) error {
	if len(rows) == 0 {
		return nil
	}
	return r.getQuery().RanFeedAdminUserRole.WithContext(r.ctx).Create(rows...)
}
