package repositories

import (
	"context"

	"gorm.io/gorm/clause"

	"ran-feed/app/rpc/count/internal/entity/model"
	"ran-feed/app/rpc/count/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
)

type BigVRepository interface {
	WithTx(tx *query.Query) BigVRepository
	Promote(userID int64, followerCount int64) error
	ListAllUserIDs() ([]int64, error)
}

type bigVRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewBigVRepository(ctx context.Context, db *orm.DB) BigVRepository {
	return &bigVRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *bigVRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *bigVRepositoryImpl) WithTx(tx *query.Query) BigVRepository {
	return &bigVRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

// Promote 大 V 晋升 粉丝数跨阈值时落表 user_id 冲突即忽略 sticky 只增不删
func (r *bigVRepositoryImpl) Promote(userID int64, followerCount int64) error {
	if userID <= 0 {
		return nil
	}

	q := r.getQuery()
	row := &model.RanFeedBigV{
		UserID:        userID,
		FollowerCount: followerCount,
	}
	return q.RanFeedBigV.WithContext(r.ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoNothing: true,
		}).
		Create(row)
}

// ListAllUserIDs 取全部大 V user_id 供全局集合周期重建 集合体量小一次取回
func (r *bigVRepositoryImpl) ListAllUserIDs() ([]int64, error) {
	q := r.getQuery()
	rows, err := q.RanFeedBigV.WithContext(r.ctx).
		Select(q.RanFeedBigV.UserID).
		Find()
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row != nil && row.UserID > 0 {
			ids = append(ids, row.UserID)
		}
	}
	return ids, nil
}
