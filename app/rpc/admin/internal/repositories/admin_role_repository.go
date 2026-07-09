package repositories

import (
	"context"
	"errors"

	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type AdminRoleRepository interface {
	WithTx(tx *query.Query) AdminRoleRepository
	// List 角色分页 id 倒序
	List(offset, limit int) ([]*model.RanFeedAdminRole, error)
	// Count 角色总数
	Count() (int64, error)
	// GetByID 按ID取角色 未命中返 nil
	GetByID(id int64) (*model.RanFeedAdminRole, error)
	// GetByCode 按 code 取角色 未命中返 nil
	GetByCode(code string) (*model.RanFeedAdminRole, error)
	// ListByIDs 按ID集合取角色 供列表富化
	ListByIDs(ids []int64) ([]*model.RanFeedAdminRole, error)
	// Create 建角色 返回自增ID
	Create(row *model.RanFeedAdminRole) (int64, error)
	// UpdateProfile 改角色名与备注 返回影响行数
	UpdateProfile(id int64, name, remark string, operatorID int64) (int64, error)
	// SoftDelete 软删角色 返回影响行数
	SoftDelete(id, operatorID int64) (int64, error)
}

type adminRoleRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewAdminRoleRepository(ctx context.Context, db *orm.DB) AdminRoleRepository {
	return &adminRoleRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *adminRoleRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *adminRoleRepositoryImpl) WithTx(tx *query.Query) AdminRoleRepository {
	return &adminRoleRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

// List 角色分页 id 倒序
func (r *adminRoleRepositoryImpl) List(offset, limit int) ([]*model.RanFeedAdminRole, error) {
	q := r.getQuery().RanFeedAdminRole
	return q.WithContext(r.ctx).Where(q.IsDeleted.Eq(0)).Order(q.ID.Desc()).Offset(offset).Limit(limit).Find()
}

// Count 角色总数
func (r *adminRoleRepositoryImpl) Count() (int64, error) {
	q := r.getQuery().RanFeedAdminRole
	return q.WithContext(r.ctx).Where(q.IsDeleted.Eq(0)).Count()
}

// GetByID 按ID取角色 未命中返 nil
func (r *adminRoleRepositoryImpl) GetByID(id int64) (*model.RanFeedAdminRole, error) {
	q := r.getQuery().RanFeedAdminRole
	row, err := q.WithContext(r.ctx).Where(q.ID.Eq(id)).Where(q.IsDeleted.Eq(0)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// GetByCode 按 code 取角色 未命中返 nil
func (r *adminRoleRepositoryImpl) GetByCode(code string) (*model.RanFeedAdminRole, error) {
	q := r.getQuery().RanFeedAdminRole
	row, err := q.WithContext(r.ctx).Where(q.Code.Eq(code)).Where(q.IsDeleted.Eq(0)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// ListByIDs 按ID集合取角色 供列表富化
func (r *adminRoleRepositoryImpl) ListByIDs(ids []int64) ([]*model.RanFeedAdminRole, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := r.getQuery().RanFeedAdminRole
	return q.WithContext(r.ctx).Where(q.ID.In(ids...)).Where(q.IsDeleted.Eq(0)).Find()
}

// Create 建角色 返回自增ID
func (r *adminRoleRepositoryImpl) Create(row *model.RanFeedAdminRole) (int64, error) {
	if row == nil {
		return 0, nil
	}
	if err := r.getQuery().RanFeedAdminRole.WithContext(r.ctx).Create(row); err != nil {
		return 0, err
	}
	return row.ID, nil
}

// UpdateProfile 改角色名与备注 返回影响行数
func (r *adminRoleRepositoryImpl) UpdateProfile(id int64, name, remark string, operatorID int64) (int64, error) {
	q := r.getQuery().RanFeedAdminRole
	res, err := q.WithContext(r.ctx).
		Where(q.ID.Eq(id)).
		Where(q.IsDeleted.Eq(0)).
		Updates(map[string]any{
			"name":       name,
			"remark":     remark,
			"updated_by": operatorID,
		})
	if err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}

// SoftDelete 软删角色 返回影响行数
func (r *adminRoleRepositoryImpl) SoftDelete(id, operatorID int64) (int64, error) {
	q := r.getQuery().RanFeedAdminRole
	res, err := q.WithContext(r.ctx).
		Where(q.ID.Eq(id)).
		Where(q.IsDeleted.Eq(0)).
		Updates(map[string]any{
			"is_deleted": 1,
			"updated_by": operatorID,
		})
	if err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}
