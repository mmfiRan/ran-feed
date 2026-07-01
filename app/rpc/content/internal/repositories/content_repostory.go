package repositories

import (
	"context"
	"errors"
	"fmt"
	"ran-feed/app/rpc/content/content"
	"time"

	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ContentRepository interface {
	WithTx(tx *query.Query) ContentRepository
	CreateContent(contentDO *do.ContentDO) error
	GetDetailByID(contentID int64) (*model.RanFeedContent, error)
	GetByIDBrief(contentID int64) (*model.RanFeedContent, error)
	DeleteByID(contentID int64) error
	GetHotScoreByID(contentID int64) (float64, error)
	CountByAuthor(status int32, visibility int32, authorID int64) (int64, error)
	ListRecommendByHotScoreCursor(status int32, visibility int32, cursorScore float64, cursorID int64, limit int) ([]*model.RanFeedContent, error)
	ListFollowByAuthorsCursor(status int32, visibility int32, authorIDs []int64, cursorMillis int64, limit int) ([]*model.RanFeedContent, error)
	ListPublishedByAuthor(authorID int64) ([]*model.RanFeedContent, error)
	ListPublishedByAuthorWithinWindow(authorID int64, sinceMillis int64, limit int) ([]*model.RanFeedContent, error)
	ListColdUpdateContents(status int32, visibility int32, start time.Time, cursorID int64, limit int) ([]*model.RanFeedContent, error)
	BatchGetRecommendByIDs(status int32, visibility int32, contentIDs []int64) (map[int64]*model.RanFeedContent, error)
	BatchGetPublishedByIDs(contentIDs []int64) (map[int64]*model.RanFeedContent, error)
	BatchGetIndexableByIDs(contentIDs []int64) (map[int64]*model.RanFeedContent, error)
	ScanIndexableByIDCursor(cursorID int64, limit int) ([]*model.RanFeedContent, error)
	BatchUpdateHotScores(ids []int64, scores []float64, updatedAt time.Time) error
}

type ContentRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewContentRepository(ctx context.Context, db *orm.DB) ContentRepository {
	return &ContentRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *ContentRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *ContentRepositoryImpl) WithTx(tx *query.Query) ContentRepository {
	return &ContentRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *ContentRepositoryImpl) CreateContent(contentDO *do.ContentDO) error {
	q := r.getQuery()

	contentModel := &model.RanFeedContent{
		ID:          contentDO.ID,
		UserID:      contentDO.UserID,
		ContentType: contentDO.ContentType,
		Status:      contentDO.Status,
		Visibility:  contentDO.Visibility,
		CreatedBy:   contentDO.CreatedBy,
		UpdatedBy:   contentDO.UpdatedBy,
		PublishedAt: contentDO.PublishedAt,
	}

	return q.RanFeedContent.WithContext(r.ctx).Create(contentModel)
}

func (r *ContentRepositoryImpl) GetDetailByID(contentID int64) (*model.RanFeedContent, error) {
	if contentID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(r.ctx).
		Select(
			q.RanFeedContent.ID,
			q.RanFeedContent.UserID,
			q.RanFeedContent.ContentType,
			q.RanFeedContent.Status,
			q.RanFeedContent.Visibility,
			q.RanFeedContent.LikeCount,
			q.RanFeedContent.FavoriteCount,
			q.RanFeedContent.CommentCount,
			q.RanFeedContent.PublishedAt,
		).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

func (r *ContentRepositoryImpl) GetByIDBrief(contentID int64) (*model.RanFeedContent, error) {
	if contentID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(r.ctx).
		Select(q.RanFeedContent.ID, q.RanFeedContent.UserID, q.RanFeedContent.ContentType).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Take()
	if err != nil {
		return nil, err
	}
	return row, nil
}

func (r *ContentRepositoryImpl) DeleteByID(contentID int64) error {
	if contentID <= 0 {
		return nil
	}

	q := r.getQuery()
	_, err := q.RanFeedContent.WithContext(r.ctx).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		UpdateSimple(q.RanFeedContent.IsDeleted.Value(1))
	return err
}

func (r *ContentRepositoryImpl) GetHotScoreByID(contentID int64) (float64, error) {
	if contentID <= 0 {
		return 0, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(r.ctx).
		Select(q.RanFeedContent.HotScore).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Take()
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, fmt.Errorf("content not found: content_id=%d", contentID)
	}
	return row.HotScore, nil
}

func (r *ContentRepositoryImpl) CountByAuthor(status int32, visibility int32, authorID int64) (int64, error) {
	if authorID <= 0 {
		return 0, nil
	}
	q := r.getQuery()
	return q.RanFeedContent.WithContext(r.ctx).
		Where(q.RanFeedContent.UserID.Eq(authorID)).
		Where(q.RanFeedContent.Status.Eq(status)).
		Where(q.RanFeedContent.Visibility.Eq(visibility)).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Where(q.RanFeedContent.PublishedAt.IsNotNull()).
		Count()
}

func (r *ContentRepositoryImpl) ListRecommendByHotScoreCursor(status int32, visibility int32, cursorScore float64, cursorID int64, limit int) ([]*model.RanFeedContent, error) {
	if limit <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	doQuery := q.RanFeedContent.WithContext(r.ctx).
		Select(q.RanFeedContent.ID, q.RanFeedContent.ContentType, q.RanFeedContent.UserID, q.RanFeedContent.PublishedAt, q.RanFeedContent.HotScore).
		Where(q.RanFeedContent.Status.Eq(status)).
		Where(q.RanFeedContent.Visibility.Eq(visibility)).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Where(q.RanFeedContent.PublishedAt.IsNotNull())

	if cursorID > 0 {
		doQuery = doQuery.
			Where(q.RanFeedContent.HotScore.Lt(cursorScore)).
			Or(q.RanFeedContent.HotScore.Eq(cursorScore), q.RanFeedContent.ID.Lt(cursorID))
	}

	rows, err := doQuery.
		Order(q.RanFeedContent.HotScore.Desc(), q.RanFeedContent.ID.Desc()).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ContentRepositoryImpl) ListFollowByAuthorsCursor(status int32, visibility int32, authorIDs []int64, cursorMillis int64, limit int) ([]*model.RanFeedContent, error) {
	if limit <= 0 {
		return nil, nil
	}
	if len(authorIDs) == 0 {
		return nil, nil
	}

	q := r.getQuery()
	doQuery := q.RanFeedContent.WithContext(r.ctx).
		Select(q.RanFeedContent.ID, q.RanFeedContent.ContentType, q.RanFeedContent.UserID, q.RanFeedContent.PublishedAt).
		Where(q.RanFeedContent.Status.Eq(status)).
		Where(q.RanFeedContent.Visibility.Eq(visibility)).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Where(q.RanFeedContent.PublishedAt.IsNotNull()).
		Where(q.RanFeedContent.UserID.In(authorIDs...))

	// 游标按 published_at 毫秒 取早于游标的内容 与 inbox/publish zset 时间序对齐
	if cursorMillis > 0 {
		doQuery = doQuery.Where(q.RanFeedContent.PublishedAt.Lt(time.UnixMilli(cursorMillis)))
	}

	rows, err := doQuery.
		Order(q.RanFeedContent.PublishedAt.Desc()).
		Order(q.RanFeedContent.ID.Desc()).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ContentRepositoryImpl) ListPublishedByAuthor(authorID int64) ([]*model.RanFeedContent, error) {
	if authorID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	return q.RanFeedContent.WithContext(r.ctx).
		Select(q.RanFeedContent.ID, q.RanFeedContent.ContentType, q.RanFeedContent.UserID, q.RanFeedContent.Visibility, q.RanFeedContent.PublishedAt).
		Where(q.RanFeedContent.UserID.Eq(authorID)).
		Where(q.RanFeedContent.Status.Eq(int32(content.ContentStatus_PUBLISHED))).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Where(q.RanFeedContent.PublishedAt.IsNotNull()).
		Order(q.RanFeedContent.PublishedAt.Desc()).
		Order(q.RanFeedContent.ID.Desc()).
		Find()
}

// ListPublishedByAuthorWithinWindow 取作者在 sinceMillis 之后发布的 PUBLIC 内容 按 published_at 倒序 capped limit
// 供关注 backfill 在 publish zset 冷时回源 只取 PUBLIC 与写扩散口径一致
func (r *ContentRepositoryImpl) ListPublishedByAuthorWithinWindow(authorID int64, sinceMillis int64, limit int) ([]*model.RanFeedContent, error) {
	if authorID <= 0 || limit <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	doQuery := q.RanFeedContent.WithContext(r.ctx).
		Select(q.RanFeedContent.ID, q.RanFeedContent.ContentType, q.RanFeedContent.UserID, q.RanFeedContent.PublishedAt).
		Where(q.RanFeedContent.UserID.Eq(authorID)).
		Where(q.RanFeedContent.Status.Eq(int32(content.ContentStatus_PUBLISHED))).
		Where(q.RanFeedContent.Visibility.Eq(int32(content.Visibility_PUBLIC))).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Where(q.RanFeedContent.PublishedAt.IsNotNull())
	if sinceMillis > 0 {
		doQuery = doQuery.Where(q.RanFeedContent.PublishedAt.Gte(time.UnixMilli(sinceMillis)))
	}
	return doQuery.
		Order(q.RanFeedContent.PublishedAt.Desc()).
		Order(q.RanFeedContent.ID.Desc()).
		Limit(limit).
		Find()
}

// ListColdUpdateContents 冷更新拉取指定时间范围内内容。
func (r *ContentRepositoryImpl) ListColdUpdateContents(status int32, visibility int32, start time.Time, cursorID int64, limit int) ([]*model.RanFeedContent, error) {
	if limit <= 0 {
		return nil, nil
	}

	q := r.getQuery()

	doQuery := q.RanFeedContent.WithContext(r.ctx).
		Select(
			q.RanFeedContent.ID,
			q.RanFeedContent.LikeCount,
			q.RanFeedContent.CommentCount,
			q.RanFeedContent.FavoriteCount,
			q.RanFeedContent.PublishedAt,
		).
		Where(q.RanFeedContent.Status.Eq(status)).
		Where(q.RanFeedContent.Visibility.Eq(visibility)).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Where(q.RanFeedContent.PublishedAt.IsNotNull()).
		Where(q.RanFeedContent.PublishedAt.Gte(start))

	if cursorID > 0 {
		doQuery = doQuery.Where(q.RanFeedContent.ID.Lt(cursorID))
	}

	return doQuery.Order(q.RanFeedContent.ID.Desc()).Limit(limit).Find()
}

func (r *ContentRepositoryImpl) BatchGetRecommendByIDs(status int32, visibility int32, contentIDs []int64) (map[int64]*model.RanFeedContent, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedContent{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedContent.WithContext(r.ctx).
		Select(q.RanFeedContent.ID, q.RanFeedContent.ContentType, q.RanFeedContent.UserID, q.RanFeedContent.PublishedAt).
		Where(q.RanFeedContent.ID.In(contentIDs...)).
		Where(q.RanFeedContent.Status.Eq(status)).
		Where(q.RanFeedContent.Visibility.Eq(visibility)).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Where(q.RanFeedContent.PublishedAt.IsNotNull()).
		Find()
	if err != nil {
		return nil, err
	}

	res := make(map[int64]*model.RanFeedContent, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		res[row.ID] = row
	}
	return res, nil
}

func (r *ContentRepositoryImpl) BatchGetPublishedByIDs(contentIDs []int64) (map[int64]*model.RanFeedContent, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedContent{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedContent.WithContext(r.ctx).
		Select(q.RanFeedContent.ID, q.RanFeedContent.ContentType, q.RanFeedContent.UserID, q.RanFeedContent.Visibility, q.RanFeedContent.PublishedAt).
		Where(q.RanFeedContent.ID.In(contentIDs...)).
		Where(q.RanFeedContent.Status.Eq(int32(content.ContentStatus_PUBLISHED))).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Where(q.RanFeedContent.PublishedAt.IsNotNull()).
		Find()
	if err != nil {
		return nil, err
	}

	res := make(map[int64]*model.RanFeedContent, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		res[row.ID] = row
	}
	return res, nil
}

// BatchGetIndexableByIDs 增量回源 取可索引内容(已发布+公开+未删除) 附建索引所需原始字段
func (r *ContentRepositoryImpl) BatchGetIndexableByIDs(contentIDs []int64) (map[int64]*model.RanFeedContent, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedContent{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedContent.WithContext(r.ctx).
		Select(
			q.RanFeedContent.ID,
			q.RanFeedContent.UserID,
			q.RanFeedContent.ContentType,
			q.RanFeedContent.Status,
			q.RanFeedContent.Visibility,
			q.RanFeedContent.HotScore,
			q.RanFeedContent.PublishedAt,
			q.RanFeedContent.UpdatedAt,
		).
		Where(q.RanFeedContent.ID.In(contentIDs...)).
		Where(q.RanFeedContent.Status.Eq(int32(content.ContentStatus_PUBLISHED))).
		Where(q.RanFeedContent.Visibility.Eq(int32(content.Visibility_PUBLIC))).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Where(q.RanFeedContent.PublishedAt.IsNotNull()).
		Find()
	if err != nil {
		return nil, err
	}

	res := make(map[int64]*model.RanFeedContent, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		res[row.ID] = row
	}
	return res, nil
}

// ScanIndexableByIDCursor 全量重建 按 id 升序 keyset 游标扫可索引内容
func (r *ContentRepositoryImpl) ScanIndexableByIDCursor(cursorID int64, limit int) ([]*model.RanFeedContent, error) {
	if limit <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	doQuery := q.RanFeedContent.WithContext(r.ctx).
		Select(
			q.RanFeedContent.ID,
			q.RanFeedContent.UserID,
			q.RanFeedContent.ContentType,
			q.RanFeedContent.Status,
			q.RanFeedContent.Visibility,
			q.RanFeedContent.HotScore,
			q.RanFeedContent.PublishedAt,
			q.RanFeedContent.UpdatedAt,
		).
		Where(q.RanFeedContent.Status.Eq(int32(content.ContentStatus_PUBLISHED))).
		Where(q.RanFeedContent.Visibility.Eq(int32(content.Visibility_PUBLIC))).
		Where(q.RanFeedContent.IsDeleted.Eq(0)).
		Where(q.RanFeedContent.PublishedAt.IsNotNull())

	if cursorID > 0 {
		doQuery = doQuery.Where(q.RanFeedContent.ID.Gt(cursorID))
	}

	return doQuery.Order(q.RanFeedContent.ID).Limit(limit).Find()
}

// BatchUpdateHotScores 批量更新热度分与更新时间。
func (r *ContentRepositoryImpl) BatchUpdateHotScores(ids []int64, scores []float64, updatedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) != len(scores) {
		return fmt.Errorf("ids and scores length mismatch")
	}
	const batchSize = 500
	q := r.getQuery()
	return q.Transaction(func(tx *query.Query) error {
		for start := 0; start < len(ids); start += batchSize {
			end := start + batchSize
			if end > len(ids) {
				end = len(ids)
			}
			for i := start; i < end; i++ {
				if _, err := tx.RanFeedContent.WithContext(r.ctx).
					Where(tx.RanFeedContent.ID.Eq(ids[i])).
					Updates(map[string]interface{}{
						"hot_score":         scores[i],
						"last_hot_score_at": updatedAt,
					}); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
