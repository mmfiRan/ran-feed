package repositories

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gen"
	"gorm.io/gorm/clause"

	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/entity/model"
	"ran-feed/app/rpc/interaction/internal/entity/query"
	"ran-feed/pkg/orm"
	"ran-feed/pkg/snowflake"
)

type FavoriteRepository interface {
	WithTx(tx *query.Query) FavoriteRepository
	CountByContentID(contentID int64) (int64, error)
	IsFavorited(userID int64, contentID int64) (bool, error)
	Upsert(favoriteDO *do.FavoriteDO) (bool, error)
	DeleteByUserAndContent(userID int64, contentID int64) (bool, error)
	ListByUserCursor(userID int64, cursor int64, limit int) ([]*model.RanFeedFavorite, error)
	GetByUserAndContent(userID int64, contentID int64) (*model.RanFeedFavorite, error)
}

type favoriteRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

const (
	favoriteStatusLegacyActive int32 = 0  // 历史数据兼容：早期写入未设置 status
	favoriteStatusActive       int32 = 10 // 正常
	favoriteStatusCanceled     int32 = 20 // 取消收藏（兼容旧软删除语义）
)

func NewFavoriteRepository(ctx context.Context, db *orm.DB) FavoriteRepository {
	return &favoriteRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *favoriteRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *favoriteRepositoryImpl) WithTx(tx *query.Query) FavoriteRepository {
	return &favoriteRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *favoriteRepositoryImpl) Upsert(favoriteDO *do.FavoriteDO) (bool, error) {
	q := r.getQuery()

	favoriteModel := &model.RanFeedFavorite{
		ID:            snowflake.GenID(),
		UserID:        favoriteDO.UserID,
		Status:        favoriteStatusActive,
		ContentID:     favoriteDO.ContentID,
		ContentUserID: favoriteDO.ContentUserID,
		CreatedBy:     favoriteDO.CreatedBy,
		UpdatedBy:     favoriteDO.UpdatedBy,
	}

	dao := q.RanFeedFavorite.WithContext(r.ctx)
	info := dao.
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "content_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"status":          favoriteStatusActive,
				"content_user_id": favoriteModel.ContentUserID,
				"updated_by":      favoriteModel.UpdatedBy,
			}),
		}).
		WithResult(func(tx gen.Dao) {
			_ = tx.Create(favoriteModel)
		})
	if info.Error != nil {
		return false, info.Error
	}
	return info.RowsAffected > 0, nil
}

func (r *favoriteRepositoryImpl) CountByContentID(contentID int64) (int64, error) {
	q := r.getQuery()
	return q.RanFeedFavorite.WithContext(r.ctx).
		Where(q.RanFeedFavorite.ContentID.Eq(contentID)).
		Where(q.RanFeedFavorite.Status.In(favoriteStatusLegacyActive, favoriteStatusActive)).
		Count()
}

func (r *favoriteRepositoryImpl) IsFavorited(userID int64, contentID int64) (bool, error) {
	q := r.getQuery()

	cnt, err := q.RanFeedFavorite.WithContext(r.ctx).
		Where(q.RanFeedFavorite.UserID.Eq(userID), q.RanFeedFavorite.ContentID.Eq(contentID)).
		Where(q.RanFeedFavorite.Status.In(favoriteStatusLegacyActive, favoriteStatusActive)).
		Count()
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (r *favoriteRepositoryImpl) DeleteByUserAndContent(userID int64, contentID int64) (bool, error) {
	q := r.getQuery()

	info, err := q.RanFeedFavorite.WithContext(r.ctx).
		Where(q.RanFeedFavorite.UserID.Eq(userID), q.RanFeedFavorite.ContentID.Eq(contentID)).
		Delete(&model.RanFeedFavorite{})
	if err != nil {
		return false, err
	}
	return info.RowsAffected > 0, nil
}

func (r *favoriteRepositoryImpl) GetByUserAndContent(userID int64, contentID int64) (*model.RanFeedFavorite, error) {
	q := r.getQuery()
	row, err := q.RanFeedFavorite.WithContext(r.ctx).
		Where(q.RanFeedFavorite.UserID.Eq(userID), q.RanFeedFavorite.ContentID.Eq(contentID)).
		Where(q.RanFeedFavorite.Status.In(favoriteStatusLegacyActive, favoriteStatusActive)).
		First()
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (r *favoriteRepositoryImpl) ListByUserCursor(userID int64, cursor int64, limit int) ([]*model.RanFeedFavorite, error) {
	if userID <= 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}

	q := r.getQuery()
	doQuery := q.RanFeedFavorite.WithContext(r.ctx).
		Where(q.RanFeedFavorite.UserID.Eq(userID)).
		Where(q.RanFeedFavorite.Status.In(favoriteStatusLegacyActive, favoriteStatusActive))

	if cursor > 0 {
		doQuery = doQuery.Where(q.RanFeedFavorite.ID.Lt(cursor))
	}

	rows, err := doQuery.
		Order(q.RanFeedFavorite.ID.Desc()).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}
