package repositories

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ran-feed/app/rpc/interaction/internal/common/enums"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/entity/model"
	"ran-feed/app/rpc/interaction/internal/entity/query"
	"ran-feed/pkg/orm"
	"ran-feed/pkg/snowflake"
)

type LikeRepository interface {
	WithTx(tx *query.Query) LikeRepository
	// ApplyLike 写入点赞记录 已存在则置为点赞态
	ApplyLike(likeDO *do.LikeDO) error
	// CancelLike 取消点赞 仅当存在且为点赞态时翻转 否则 no-op
	CancelLike(likeDO *do.LikeDO) error
	// BatchUpsert 批量插入或更新点赞记录
	BatchUpsert(likeDOs []*do.LikeDO) error
	// GetByUserAndContent 根据用户ID和内容ID查询点赞记录
	GetByUserAndContent(userID, contentID int64) (*do.LikeDO, error)
	// IsLiked 判断用户是否已点赞
	IsLiked(userID, contentID int64) (bool, error)
	// BatchIsLiked 批量判断用户是否已点赞（返回 content_id -> is_liked）
	BatchIsLiked(userID int64, contentIDs []int64) (map[int64]bool, error)
	// QueryUserLikedTopN 查询用户最新 N 条有效点赞 按 content_id 降序 用于缓存重建
	QueryUserLikedTopN(userID int64, limit int) ([]int64, error)
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

// ApplyLike 写入点赞记录 已存在则置为点赞态
// 重复点赞时 status 值未变 InnoDB 不写 binlog updated_by 也仅在状态翻转时更新
func (r *likeRepositoryImpl) ApplyLike(likeDO *do.LikeDO) error {
	q := r.getQuery()
	db := q.RanFeedLike.WithContext(r.ctx).UnderlyingDB()
	sql := `
INSERT INTO ran_feed_like
	(id, user_id, content_id, content_user_id, status, created_by, updated_by)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	status          = VALUES(status),
	updated_by      = IF(status <> VALUES(status), VALUES(updated_by), updated_by),
	content_user_id = IF(content_user_id = 0, VALUES(content_user_id), content_user_id)
`
	res := db.WithContext(r.ctx).Exec(
		sql,
		snowflake.GenID(),
		likeDO.UserID,
		likeDO.ContentID,
		likeDO.ContentUserID,
		enums.LikeStatusLike.Int32(),
		likeDO.CreatedBy,
		likeDO.UpdatedBy,
	)
	return res.Error
}

// CancelLike 取消点赞 仅当 (user_id, content_id) 已存在且当前为点赞态时翻转
func (r *likeRepositoryImpl) CancelLike(likeDO *do.LikeDO) error {
	q := r.getQuery()
	db := q.RanFeedLike.WithContext(r.ctx).UnderlyingDB()
	sql := `
UPDATE ran_feed_like
SET status = ?, updated_by = ?
WHERE user_id = ? AND content_id = ? AND status = ?
`
	res := db.WithContext(r.ctx).Exec(
		sql,
		enums.LikeStatusCancel.Int32(),
		likeDO.UpdatedBy,
		likeDO.UserID,
		likeDO.ContentID,
		enums.LikeStatusLike.Int32(),
	)
	return res.Error
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
		Status:        enums.LikeStatus(likeModel.Status),
		CreatedBy:     likeModel.CreatedBy,
		UpdatedBy:     likeModel.UpdatedBy,
	}, nil
}

func (r *likeRepositoryImpl) IsLiked(userID, contentID int64) (bool, error) {
	q := r.getQuery()

	count, err := q.RanFeedLike.WithContext(r.ctx).
		Where(q.RanFeedLike.UserID.Eq(userID)).
		Where(q.RanFeedLike.ContentID.Eq(contentID)).
		Where(q.RanFeedLike.Status.Eq(enums.LikeStatusLike.Int32())).
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
		Where(q.RanFeedLike.Status.Eq(enums.LikeStatusLike.Int32())).
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
			Status:        likeDO.Status.Int32(),
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

// QueryUserLikedTopN 查询用户最新 N 条有效点赞 按 content_id 降序
func (r *likeRepositoryImpl) QueryUserLikedTopN(userID int64, limit int) ([]int64, error) {
	if userID <= 0 || limit <= 0 {
		return []int64{}, nil
	}

	q := r.getQuery()
	var contentIDs []int64
	err := q.RanFeedLike.WithContext(r.ctx).
		Select(q.RanFeedLike.ContentID).
		Where(q.RanFeedLike.UserID.Eq(userID)).
		Where(q.RanFeedLike.Status.Eq(enums.LikeStatusLike.Int32())).
		Order(q.RanFeedLike.ContentID.Desc()).
		Limit(limit).
		Pluck(q.RanFeedLike.ContentID, &contentIDs)
	if err != nil {
		return nil, err
	}
	return contentIDs, nil
}

// GetLikedUserIDs 获取内容的所有点赞用户ID列表
func (r *likeRepositoryImpl) GetLikedUserIDs(contentID int64) ([]int64, error) {
	q := r.getQuery()

	// 查询所有点赞状态为10的用户ID
	var userIDs []int64
	err := q.RanFeedLike.WithContext(r.ctx).
		Select(q.RanFeedLike.UserID).
		Where(q.RanFeedLike.ContentID.Eq(contentID)).
		Where(q.RanFeedLike.Status.Eq(enums.LikeStatusLike.Int32())).
		Pluck(q.RanFeedLike.UserID, &userIDs)

	if err != nil {
		return nil, err
	}

	return userIDs, nil
}
