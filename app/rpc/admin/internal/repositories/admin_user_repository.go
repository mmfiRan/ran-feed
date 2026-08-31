package repositories

import (
	"context"
	"ran-feed/pkg/enum"

	"gorm.io/gorm"

	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUserRepository interface {
	WithTx(tx *query.Query) AdminUserRepository
	GetByUsername(username string) (*model.RanFeedAdminUser, error)
	GetByID(id int64) (*model.RanFeedAdminUser, error)
	// List 管理员分页 status>0 时按状态筛选 id 倒序
	List(status int32, offset, limit int) ([]*model.RanFeedAdminUser, error)
	// Count 管理员总数 status>0 时按状态筛选
	Count(status int32) (int64, error)
	// Create 建管理员
	Create(row *model.RanFeedAdminUser) error
	// UpdateProfile 改昵称 返回影响行数
	UpdateProfile(id int64, nickname string, operatorID int64) (int64, error)
	// UpdateStatus 改状态 返回影响行数
	UpdateStatus(id int64, status int32, operatorID int64) (int64, error)
	// UpdatePassword 改密码哈希 返回影响行数
	UpdatePassword(id int64, hash string, operatorID int64) (int64, error)
}

type adminUserRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewAdminUserRepository(ctx context.Context, db *orm.DB) AdminUserRepository {
	return &adminUserRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *adminUserRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *adminUserRepositoryImpl) WithTx(tx *query.Query) AdminUserRepository {
	return &adminUserRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

// GetByUsername 按用户名取管理员 未命中返回 nil
func (r *adminUserRepositoryImpl) GetByUsername(username string) (*model.RanFeedAdminUser, error) {
	q := r.getQuery().RanFeedAdminUser
	row, err := q.WithContext(r.ctx).
		Where(q.Username.Eq(username)).
		Where(q.IsDeleted.Eq(enum.NotDeleted.Int32())).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// GetByID 按主键取管理员 未命中返回 nil
func (r *adminUserRepositoryImpl) GetByID(id int64) (*model.RanFeedAdminUser, error) {
	q := r.getQuery().RanFeedAdminUser
	row, err := q.WithContext(r.ctx).
		Where(q.ID.Eq(id)).
		Where(q.IsDeleted.Eq(0)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// List 管理员分页 status>0 时按状态筛选 id 倒序
func (r *adminUserRepositoryImpl) List(status int32, offset, limit int) ([]*model.RanFeedAdminUser, error) {
	q := r.getQuery().RanFeedAdminUser
	do := q.WithContext(r.ctx).Where(q.IsDeleted.Eq(0))
	if status > 0 {
		do = do.Where(q.Status.Eq(status))
	}
	return do.Order(q.ID.Desc()).Offset(offset).Limit(limit).Find()
}

// Count 管理员总数 status>0 时按状态筛选
func (r *adminUserRepositoryImpl) Count(status int32) (int64, error) {
	q := r.getQuery().RanFeedAdminUser
	do := q.WithContext(r.ctx).Where(q.IsDeleted.Eq(0))
	if status > 0 {
		do = do.Where(q.Status.Eq(status))
	}
	return do.Count()
}

// Create 建管理员 返回ID
func (r *adminUserRepositoryImpl) Create(row *model.RanFeedAdminUser) error {
	if row == nil {
		return nil
	}
	if err := r.getQuery().RanFeedAdminUser.WithContext(r.ctx).Create(row); err != nil {
		return err
	}
	return nil
}

// UpdateProfile 改昵称 返回影响行数
func (r *adminUserRepositoryImpl) UpdateProfile(id int64, nickname string, operatorID int64) (int64, error) {
	q := r.getQuery().RanFeedAdminUser
	res, err := q.WithContext(r.ctx).
		Where(q.ID.Eq(id)).
		Where(q.IsDeleted.Eq(0)).
		Updates(map[string]any{"nickname": nickname, "updated_by": operatorID})
	if err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}

// UpdateStatus 改状态 返回影响行数
func (r *adminUserRepositoryImpl) UpdateStatus(id int64, status int32, operatorID int64) (int64, error) {
	q := r.getQuery().RanFeedAdminUser
	res, err := q.WithContext(r.ctx).
		Where(q.ID.Eq(id)).
		Where(q.IsDeleted.Eq(0)).
		Updates(map[string]any{"status": status, "updated_by": operatorID})
	if err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}

// UpdatePassword 改密码哈希 返回影响行数
func (r *adminUserRepositoryImpl) UpdatePassword(id int64, hash string, operatorID int64) (int64, error) {
	q := r.getQuery().RanFeedAdminUser
	res, err := q.WithContext(r.ctx).
		Where(q.ID.Eq(id)).
		Where(q.IsDeleted.Eq(0)).
		Updates(map[string]any{"password_hash": hash, "updated_by": operatorID})
	if err != nil {
		return 0, err
	}
	return res.RowsAffected, nil
}
