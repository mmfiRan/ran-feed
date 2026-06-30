package repositories

import (
	"context"

	"ran-feed/app/rpc/search/internal/common/consts"
	"ran-feed/app/rpc/search/internal/entity/model"
	"ran-feed/app/rpc/search/internal/entity/query"
	"ran-feed/pkg/enum"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserRepository interface {
	ScanActive(cursorID int64, limit int) ([]*model.RanFeedUser, error)
	GetByIDs(ids []int64) (map[int64]*model.RanFeedUser, error)
}

type userRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
}

func NewUserRepository(ctx context.Context, db *orm.DB) UserRepository {
	return &userRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

// ScanActive 按 id 游标批量扫正常未删除用户 供全量重建
func (r *userRepositoryImpl) ScanActive(cursorID int64, limit int) ([]*model.RanFeedUser, error) {
	if limit <= 0 {
		return nil, nil
	}

	q := query.Q
	rows, err := q.RanFeedUser.WithContext(r.ctx).
		Where(q.RanFeedUser.IsDeleted.Eq(enum.NotDeleted.Int32())).
		Where(q.RanFeedUser.Status.Eq(consts.UserStatusNormal)).
		Where(q.RanFeedUser.ID.Gt(cursorID)).
		Order(q.RanFeedUser.ID).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetByIDs 按 id 批量取未删除用户 软删行不返回 供消费侧判 upsert 或 delete
func (r *userRepositoryImpl) GetByIDs(ids []int64) (map[int64]*model.RanFeedUser, error) {
	res := make(map[int64]*model.RanFeedUser, len(ids))
	if len(ids) == 0 {
		return res, nil
	}

	q := query.Q
	rows, err := q.RanFeedUser.WithContext(r.ctx).
		Where(q.RanFeedUser.IsDeleted.Eq(enum.NotDeleted.Int32())).
		Where(q.RanFeedUser.ID.In(ids...)).
		Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row == nil {
			continue
		}
		res[row.ID] = row
	}
	return res, nil
}
