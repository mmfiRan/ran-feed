// Package grouprepo 群元信息读写
package grouprepo

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/pkg/enums"
)

var _ repositories.GroupRepository = (*Repository)(nil)

// Repository 群仓储
type Repository struct {
	tx *query.Query
}

// New 构造仓储
func New() *Repository {
	return &Repository{}
}

func (r *Repository) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

// WithTx 绑定事务
func (r *Repository) WithTx(tx *query.Query) repositories.GroupRepository {
	return &Repository{tx: tx}
}

// Create 新建群
func (r *Repository) Create(ctx context.Context, row *model.RanFeedImGroup) error {
	if row == nil {
		return nil
	}
	return r.getQuery().RanFeedImGroup.WithContext(ctx).Create(row)
}

// GetByID 按 ID 查群
func (r *Repository) GetByID(ctx context.Context, groupID int64) (*model.RanFeedImGroup, error) {
	if groupID <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedImGroup.WithContext(ctx).
		Where(q.RanFeedImGroup.ID.Eq(groupID)).
		Where(q.RanFeedImGroup.IsDeleted.Eq(enums.NotDeleted.Int32())).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// GetByIDForUpdate 行锁查群
func (r *Repository) GetByIDForUpdate(ctx context.Context, groupID int64) (*model.RanFeedImGroup, error) {
	if groupID <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedImGroup.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(q.RanFeedImGroup.ID.Eq(groupID)).
		Where(q.RanFeedImGroup.IsDeleted.Eq(enums.NotDeleted.Int32())).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// BatchGetByIDs 批量取群
func (r *Repository) BatchGetByIDs(ctx context.Context, groupIDs []int64) (map[int64]*model.RanFeedImGroup, error) {
	out := make(map[int64]*model.RanFeedImGroup, len(groupIDs))
	if len(groupIDs) == 0 {
		return out, nil
	}
	q := r.getQuery()
	rows, err := q.RanFeedImGroup.WithContext(ctx).
		Where(q.RanFeedImGroup.ID.In(groupIDs...)).
		Where(q.RanFeedImGroup.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

// UpdateMemberCount 更新成员数
func (r *Repository) UpdateMemberCount(ctx context.Context, groupID int64, memberCount int32, updatedBy int64) error {
	if groupID <= 0 {
		return nil
	}
	q := r.getQuery()
	_, err := q.RanFeedImGroup.WithContext(ctx).
		Where(q.RanFeedImGroup.ID.Eq(groupID)).
		Where(q.RanFeedImGroup.IsDeleted.Eq(enums.NotDeleted.Int32())).
		UpdateSimple(
			q.RanFeedImGroup.MemberCount.Value(memberCount),
			q.RanFeedImGroup.UpdatedBy.Value(updatedBy),
		)
	return err
}

// Dissolve 解散群
func (r *Repository) Dissolve(ctx context.Context, groupID, updatedBy int64) error {
	if groupID <= 0 {
		return nil
	}
	q := r.getQuery()
	_, err := q.RanFeedImGroup.WithContext(ctx).
		Where(q.RanFeedImGroup.ID.Eq(groupID)).
		Where(q.RanFeedImGroup.Status.Eq(imEnum.GroupStatusNormal.Int32())).
		Where(q.RanFeedImGroup.IsDeleted.Eq(enums.NotDeleted.Int32())).
		UpdateSimple(
			q.RanFeedImGroup.Status.Value(imEnum.GroupStatusDissolved.Int32()),
			q.RanFeedImGroup.UpdatedBy.Value(updatedBy),
		)
	return err
}
