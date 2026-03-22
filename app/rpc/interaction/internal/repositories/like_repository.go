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
	LikeStatusLike   int32 = 10 // 点赞
	LikeStatusCancel int32 = 20 // 取消
)

type LikeRepository interface {
	WithTx(tx *query.Query) LikeRepository
	// Upsert 插入或更新点赞记录
	Upsert(likeDO *do.LikeDO) error
	// BatchUpsert 批量插入或更新点赞记录
	BatchUpsert(likeDOs []*do.LikeDO) error
	// GetByUserAndContent 根据用户ID和内容ID查询点赞记录
	GetByUserAndContent(userID, contentID int64) (*do.LikeDO, error)
	// IsLiked 判断用户是否已点赞
	IsLiked(userID, contentID int64) (bool, error)
	// BatchIsLiked 批量判断用户是否已点赞（返回 content_id -> is_liked）
	BatchIsLiked(userID int64, contentIDs []int64) (map[int64]bool, error)
	// GetLikedUserIDs 获取内容的所有点赞用户ID列表
	GetLikedUserIDs(contentID int64) ([]int64, error)
}

type likeRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewLikeRepository(ctx context.Context, db *orm.DB) LikeRepository {
	return &likeRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *likeRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *likeRepositoryImpl) WithTx(tx *query.Query) LikeRepository {
	return &likeRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *likeRepositoryImpl) Upsert(likeDO *do.LikeDO) error {
	q := r.getQuery()
	db := q.RanFeedLike.WithContext(r.ctx).UnderlyingDB()
	sql := `
INSERT INTO ran_feed_like
	(id, user_id, content_id, content_user_id, status, created_by, updated_by)
SELECT ?, ?, ?, ?, ?, ?, ?
FROM dual
WHERE ? = ?
   OR EXISTS (
	   SELECT 1
	   FROM ran_feed_like
	   WHERE user_id = ? AND content_id = ?
   )
ON DUPLICATE KEY UPDATE
	status = IF(status <> VALUES(status), VALUES(status), status),
	updated_by = IF(status <> VALUES(status), VALUES(updated_by), updated_by),
	content_user_id = IF(content_user_id = 0, VALUES(content_user_id), content_user_id)
`
	res := db.WithContext(r.ctx).Exec(
		sql,
		snowflake.GenID(),
		likeDO.UserID,
		likeDO.ContentID,
		likeDO.ContentUserID,
		likeDO.Status,
		likeDO.CreatedBy,
		likeDO.UpdatedBy,
		likeDO.Status,
		LikeStatusLike,
		likeDO.UserID,
		likeDO.ContentID,
	)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *likeRepositoryImpl) GetByUserAndContent(userID, contentID int64) (*do.LikeDO, error) {
	q := r.getQuery()

	likeModel, err := q.RanFeedLike.WithContext(r.ctx).
		Where(q.RanFeedLike.UserID.Eq(userID)).
		Where(q.RanFeedLike.ContentID.Eq(contentID)).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &do.LikeDO{
		ID:            likeModel.ID,
		UserID:        likeModel.UserID,
		ContentID:     likeModel.ContentID,
		ContentUserID: likeModel.ContentUserID,
		Status:        likeModel.Status,
		CreatedBy:     likeModel.CreatedBy,
		UpdatedBy:     likeModel.UpdatedBy,
	}, nil
}

func (r *likeRepositoryImpl) IsLiked(userID, contentID int64) (bool, error) {
	q := r.getQuery()

	count, err := q.RanFeedLike.WithContext(r.ctx).
		Where(q.RanFeedLike.UserID.Eq(userID)).
		Where(q.RanFeedLike.ContentID.Eq(contentID)).
		Where(q.RanFeedLike.Status.Eq(LikeStatusLike)).
		Count()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *likeRepositoryImpl) BatchIsLiked(userID int64, contentIDs []int64) (map[int64]bool, error) {
	result := make(map[int64]bool, len(contentIDs))
	if userID <= 0 || len(contentIDs) == 0 {
		return result, nil
	}

	unique := make([]int64, 0, len(contentIDs))
	seen := make(map[int64]struct{}, len(contentIDs))
	for _, contentID := range contentIDs {
		if contentID <= 0 {
			continue
		}
		if _, ok := seen[contentID]; ok {
			continue
		}
		seen[contentID] = struct{}{}
		unique = append(unique, contentID)
	}
	if len(unique) == 0 {
		return result, nil
	}

	q := r.getQuery()
	var likedContentIDs []int64
	err := q.RanFeedLike.WithContext(r.ctx).
		Select(q.RanFeedLike.ContentID).
		Where(q.RanFeedLike.UserID.Eq(userID)).
		Where(q.RanFeedLike.ContentID.In(unique...)).
		Where(q.RanFeedLike.Status.Eq(LikeStatusLike)).
		Pluck(q.RanFeedLike.ContentID, &likedContentIDs)
	if err != nil {
		return nil, err
	}
	for _, contentID := range likedContentIDs {
		result[contentID] = true
	}
	return result, nil
}

// BatchUpsert 批量插入或更新点赞记录
func (r *likeRepositoryImpl) BatchUpsert(likeDOs []*do.LikeDO) error {
	if len(likeDOs) == 0 {
		return nil
	}

	q := r.getQuery()
	likeModels := make([]*model.RanFeedLike, 0, len(likeDOs))

	for _, likeDO := range likeDOs {
		likeModels = append(likeModels, &model.RanFeedLike{
			ID:            snowflake.GenID(),
			UserID:        likeDO.UserID,
			ContentID:     likeDO.ContentID,
			ContentUserID: likeDO.ContentUserID,
			Status:        likeDO.Status,
			CreatedBy:     likeDO.CreatedBy,
			UpdatedBy:     likeDO.UpdatedBy,
		})
	}

	// 一条 SQL 批量插入，冲突时更新 status 和 updated_by
	return q.RanFeedLike.WithContext(r.ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "content_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "updated_by", "content_user_id"}),
		}).
		CreateInBatches(likeModels, len(likeModels))
}

// GetLikedUserIDs 获取内容的所有点赞用户ID列表
func (r *likeRepositoryImpl) GetLikedUserIDs(contentID int64) ([]int64, error) {
	q := r.getQuery()

	// 查询所有点赞状态为10的用户ID
	var userIDs []int64
	err := q.RanFeedLike.WithContext(r.ctx).
		Select(q.RanFeedLike.UserID).
		Where(q.RanFeedLike.ContentID.Eq(contentID)).
		Where(q.RanFeedLike.Status.Eq(LikeStatusLike)).
		Pluck(q.RanFeedLike.UserID, &userIDs)

	if err != nil {
		return nil, err
	}

	return userIDs, nil
}
