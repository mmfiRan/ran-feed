package repositories

import (
	"context"
	"ran-feed/pkg/enums"

	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminPermissionRepository interface {
	ListCodesByIDs(ids []int64) ([]string, error)
	// ListCodesByAdminID 查询权限集合
	ListCodesByAdminID(adminID int64) ([]string, error)
	// ListAll 取权限点目录
	ListAll(module string) ([]*model.RanFeedAdminPermission, error)
}

type adminPermissionRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewAdminPermissionRepository(ctx context.Context, db *orm.DB) AdminPermissionRepository {
	return &adminPermissionRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// ListCodesByIDs 按权限点ID集合取 code 集合
func (r *adminPermissionRepositoryImpl) ListCodesByIDs(ids []int64) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := query.Q
	rows, err := q.RanFeedAdminPermission.WithContext(r.ctx).
		Select(q.RanFeedAdminPermission.Code).
		Where(q.RanFeedAdminPermission.ID.In(ids...)).
		Where(q.RanFeedAdminPermission.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Find()
	if err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.Code != "" {
			codes = append(codes, row.Code)
		}
	}
	return codes, nil
}

// ListCodesByAdminID 查询权限集合
func (r *adminPermissionRepositoryImpl) ListCodesByAdminID(adminID int64) ([]string, error) {
	ur := query.Q.RanFeedAdminUserRole
	rp := query.Q.RanFeedAdminRolePermission
	p := query.Q.RanFeedAdminPermission

	codes := make([]string, 0)
	err := ur.WithContext(r.ctx).
		Select(p.Code).
		Join(&model.RanFeedAdminRolePermission{}, ur.RoleID.EqCol(rp.RoleID), rp.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Join(&model.RanFeedAdminPermission{}, rp.PermissionID.EqCol(p.ID), p.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Where(ur.AdminUserID.Eq(adminID), ur.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Distinct().
		Scan(&codes)
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// ListAll 取权限点目录
func (r *adminPermissionRepositoryImpl) ListAll(module string) ([]*model.RanFeedAdminPermission, error) {
	q := query.Q.RanFeedAdminPermission
	do := q.WithContext(r.ctx).Where(q.IsDeleted.Eq(enums.NotDeleted.Int32()))
	if module != "" {
		do = do.Where(q.Module.Eq(module))
	}
	return do.Order(q.Module, q.ID).Find()
}
