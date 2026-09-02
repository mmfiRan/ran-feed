package repositories

import (
	"context"
	"errors"
	"ran-feed/pkg/enums"

	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type AdminRoleRepository interface {
	WithTx(tx *query.Query) AdminRoleRepository
	// Page 角色分页 id 倒序 返回列表与总数
	Page(offset, limit int) ([]*model.RanFeedAdminRole, int64, error)
	// ListByAdminID 取管理员绑定的角色
	ListByAdminID(adminID int64) ([]*model.RanFeedAdminRole, error)
	// GetByID 按ID取角色 未命中返 nil
	GetByID(id int64) (*model.RanFeedAdminRole, error)
	// GetByCode 按 code 取角色 未命中返 nil
	GetByCode(code string) (*model.RanFeedAdminRole, error)
	// ListByIDs 按ID集合取角色
	ListByIDs(ids []int64) ([]*model.RanFeedAdminRole, error)
	// Create 建角色
	Create(row *model.RanFeedAdminRole) error
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

// Page 角色分页 id 倒序 复用 gen FindByPage 末页不满免 COUNT
func (r *adminRoleRepositoryImpl) Page(offset, limit int) ([]*model.RanFeedAdminRole, int64, error) {
	q := r.getQuery().RanFeedAdminRole
	return q.WithContext(r.ctx).
		Where(q.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Order(q.ID.Desc()).
		FindByPage(offset, limit)
}

// ListByAdminID 取管理员绑定的角色
func (r *adminRoleRepositoryImpl) ListByAdminID(adminID int64) ([]*model.RanFeedAdminRole, error) {
	role := r.getQuery().RanFeedAdminRole
	ur := r.getQuery().RanFeedAdminUserRole
	rows := make([]*model.RanFeedAdminRole, 0)
	err := role.WithContext(r.ctx).
		Select(role.ID, role.Code).
		LeftJoin(ur, ur.RoleID.EqCol(role.ID)).
		Where(ur.AdminUserID.Eq(adminID)).
		Where(ur.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Where(role.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Scan(&rows)
	if err != nil {
		return nil, err
	}
	out := make([]*model.RanFeedAdminRole, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.ID > 0 {
			out = append(out, row)
		}
	}
	return out, nil
}

// GetByID 按ID取角色 未命中返 nil
func (r *adminRoleRepositoryImpl) GetByID(id int64) (*model.RanFeedAdminRole, error) {
	q := r.getQuery().RanFeedAdminRole
	row, err := q.WithContext(r.ctx).
		Where(q.ID.Eq(id)).
		Where(q.IsDeleted.Eq(enums.NotDeleted.Int32())).
		First()
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
	row, err := q.WithContext(r.ctx).
		Where(q.Code.Eq(code)).
		Where(q.IsDeleted.Eq(enums.NotDeleted.Int32())).
		First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// ListByIDs 按ID集合取角色
func (r *adminRoleRepositoryImpl) ListByIDs(ids []int64) ([]*model.RanFeedAdminRole, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	q := r.getQuery().RanFeedAdminRole
	return q.WithContext(r.ctx).
		Where(q.ID.In(ids...)).
		Where(q.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Find()
}

// Create 建角色
func (r *adminRoleRepositoryImpl) Create(row *model.RanFeedAdminRole) error {
	if row == nil {
		return nil
	}
	if err := r.getQuery().RanFeedAdminRole.WithContext(r.ctx).Create(row); err != nil {
		return err
	}
	return nil
}

// UpdateProfile 改角色名与备注 返回影响行数
func (r *adminRoleRepositoryImpl) UpdateProfile(id int64, name, remark string, operatorID int64) (int64, error) {
	q := r.getQuery().RanFeedAdminRole
	res, err := q.WithContext(r.ctx).
		Where(q.ID.Eq(id)).
		Where(q.IsDeleted.Eq(enums.NotDeleted.Int32())).
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
		Where(q.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Updates(map[string]any{
			"is_deleted": 1,
			"updated_by": operatorID,
		})
	if err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}
