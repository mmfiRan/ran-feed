package repositories

import (
	"context"

	"gorm.io/gorm"

	"ran-feed/app/rpc/admin/internal/entity/model"
	"ran-feed/app/rpc/admin/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminUserRepository interface {
	GetByUsername(username string) (*model.RanFeedAdminUser, error)
	GetByID(id int64) (*model.RanFeedAdminUser, error)
}

type adminUserRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewAdminUserRepository(ctx context.Context, db *orm.DB) AdminUserRepository {
	return &adminUserRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// GetByUsername 按用户名取管理员 未命中返回 nil
func (r *adminUserRepositoryImpl) GetByUsername(username string) (*model.RanFeedAdminUser, error) {
	q := query.Q
	row, err := q.RanFeedAdminUser.WithContext(r.ctx).
		Where(q.RanFeedAdminUser.Username.Eq(username)).
		Where(q.RanFeedAdminUser.IsDeleted.Eq(0)).
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
	q := query.Q
	row, err := q.RanFeedAdminUser.WithContext(r.ctx).
		Where(q.RanFeedAdminUser.ID.Eq(id)).
		Where(q.RanFeedAdminUser.IsDeleted.Eq(0)).
		First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}
