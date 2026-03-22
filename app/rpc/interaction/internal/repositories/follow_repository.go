package repositories

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/entity/model"
	"ran-feed/app/rpc/interaction/internal/entity/query"
	"ran-feed/pkg/orm"
	"ran-feed/pkg/snowflake"
)

const (
	FollowStatusFollow   int32 = 10 // 关注
	FollowStatusUnfollow int32 = 20 // 取消关注
)

type FollowRepository interface {
	WithTx(tx *query.Query) FollowRepository
	Upsert(followDO *do.FollowDO) error
	GetByUserAndFollow(userID, followUserID int64) (*do.FollowDO, error)
	IsFollowing(userID, followUserID int64) (bool, error)
	CountFollowees(userID int64) (int64, error)
	CountFollowers(userID int64) (int64, error)
	ListFolloweesByCursor(userID int64, cursorFollowUserID int64, limit int) ([]int64, error)
}

type followRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewFollowRepository(ctx context.Context, db *orm.DB) FollowRepository {
	return &followRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *followRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *followRepositoryImpl) WithTx(tx *query.Query) FollowRepository {
	return &followRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *followRepositoryImpl) Upsert(followDO *do.FollowDO) error {
	q := r.getQuery()
	row := &model.RanFeedFollow{
		ID:           snowflake.GenID(),
		UserID:       followDO.UserID,
		FollowUserID: followDO.FollowUserID,
		Status:       followDO.Status,
		CreatedBy:    followDO.CreatedBy,
		UpdatedBy:    followDO.UpdatedBy,
	}

	return q.RanFeedFollow.WithContext(r.ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "follow_user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "updated_by"}),
		}).
		Create(row)
}

func (r *followRepositoryImpl) GetByUserAndFollow(userID, followUserID int64) (*do.FollowDO, error) {
	q := r.getQuery()
	row, err := q.RanFeedFollow.WithContext(r.ctx).
		Where(q.RanFeedFollow.UserID.Eq(userID)).
		Where(q.RanFeedFollow.FollowUserID.Eq(followUserID)).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &do.FollowDO{
		UserID:       row.UserID,
		FollowUserID: row.FollowUserID,
		Status:       row.Status,
		CreatedBy:    row.CreatedBy,
		UpdatedBy:    row.UpdatedBy,
	}, nil
}

func (r *followRepositoryImpl) IsFollowing(userID, followUserID int64) (bool, error) {
	if userID <= 0 || followUserID <= 0 {
		return false, nil
	}
	row, err := r.GetByUserAndFollow(userID, followUserID)
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}
	return row.Status == FollowStatusFollow, nil
}

func (r *followRepositoryImpl) CountFollowees(userID int64) (int64, error) {
	if userID <= 0 {
		return 0, nil
	}
	q := r.getQuery()
	return q.RanFeedFollow.WithContext(r.ctx).
		Where(q.RanFeedFollow.UserID.Eq(userID)).
		Where(q.RanFeedFollow.Status.Eq(FollowStatusFollow)).
		Where(q.RanFeedFollow.IsDeleted.Eq(0)).
		Count()
}

func (r *followRepositoryImpl) CountFollowers(userID int64) (int64, error) {
	if userID <= 0 {
		return 0, nil
	}
	q := r.getQuery()
	return q.RanFeedFollow.WithContext(r.ctx).
		Where(q.RanFeedFollow.FollowUserID.Eq(userID)).
		Where(q.RanFeedFollow.Status.Eq(FollowStatusFollow)).
		Where(q.RanFeedFollow.IsDeleted.Eq(0)).
		Count()
}

func (r *followRepositoryImpl) ListFolloweesByCursor(userID int64, cursorFollowUserID int64, limit int) ([]int64, error) {
	if userID <= 0 || limit <= 0 {
		return []int64{}, nil
	}

	q := r.getQuery()
	doQuery := q.RanFeedFollow.WithContext(r.ctx).
		Select(q.RanFeedFollow.FollowUserID).
		Where(q.RanFeedFollow.UserID.Eq(userID)).
		Where(q.RanFeedFollow.Status.Eq(FollowStatusFollow)).
		Where(q.RanFeedFollow.IsDeleted.Eq(0))

	if cursorFollowUserID > 0 {
		doQuery = doQuery.Where(q.RanFeedFollow.FollowUserID.Lt(cursorFollowUserID))
	}

	rows, err := doQuery.
		Order(q.RanFeedFollow.FollowUserID.Desc()).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		ids = append(ids, row.FollowUserID)
	}
	return ids, nil
}
