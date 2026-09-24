package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/pkg/enums"
	"ran-feed/pkg/orm"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
)

// hotScoreBatchSize 单条批量更新语句覆盖的行数 防 SQL 过长
const hotScoreBatchSize = 500

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
	ListPublishedByAuthorWithinWindow(authorID int64, sinceMillis int64, limit int) ([]*model.RanFeedContent, error)
	ListColdUpdateContents(status int32, visibility int32, start time.Time, cursorID int64, limit int) ([]*model.RanFeedContent, error)
	BatchGetRecommendByIDs(status int32, visibility int32, contentIDs []int64) (map[int64]*model.RanFeedContent, error)
	BatchGetPublishedByIDs(contentIDs []int64) (map[int64]*model.RanFeedContent, error)
	BatchGetIndexableByIDs(contentIDs []int64) (map[int64]*model.RanFeedContent, error)
	ScanIndexableByIDCursor(cursorID int64, limit int) ([]*model.RanFeedContent, error)
	BatchUpdateHotScores(ids []int64, scores []float64, updatedAt time.Time) error
	// AdminPageContents 管理端多条件筛选
	AdminPageContents(status *int32, contentType *int32, authorID *int64, offset, limit int) ([]*model.RanFeedContent, int64, error)
	// AdminGetByID 管理端取任意状态内容
	AdminGetByID(contentID int64) (*model.RanFeedContent, error)
	// AdminUpdateStatus 管理端条件翻转状态
	AdminUpdateStatus(contentID int64, fromStatus, toStatus int32, operatorID int64) (int64, error)
	// AdminApproveContent 审核通过 PENDING_REVIEW->PUBLISHED
	AdminApproveContent(contentID, operatorID int64, publishedAt time.Time) (int64, error)
	// GetOwnedByID 取本人任意状态内容 草稿编辑与提交鉴权用
	GetOwnedByID(contentID, userID int64) (*model.RanFeedContent, error)
	// UpdateDraftMeta 草稿编辑更新内容级字段 状态回落 DRAFT
	UpdateDraftMeta(contentID int64, visibility int32, updatedBy int64) error
	// SubmitOwned 本人指定状态提交进审核 条件更新返回受影响行数
	SubmitOwned(contentID, userID int64, fromStatuses []int32, toStatus int32, updatedBy int64) (int64, error)
	// MyContentPage 我的内容 全状态 keyset 游标按 id 倒序 id 与 created_at 同序
	MyContentPage(userID int64, status *int32, contentType *int32, cursorID int64, limit int) ([]*model.RanFeedContent, error)
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
	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(r.ctx).
		Select(
			q.RanFeedContent.ID,
			q.RanFeedContent.UserID,
			q.RanFeedContent.ContentType,
			q.RanFeedContent.Status,
			q.RanFeedContent.Visibility,
			q.RanFeedContent.PublishedAt,
		).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
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
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

func (r *ContentRepositoryImpl) DeleteByID(contentID int64) error {
	q := r.getQuery()
	_, err := q.RanFeedContent.WithContext(r.ctx).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		UpdateSimple(q.RanFeedContent.IsDeleted.Value(enums.Deleted.Int32()))
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
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
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
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Where(q.RanFeedContent.PublishedAt.IsNotNull())

	if cursorID > 0 {
		doQuery = doQuery.Where(
			q.RanFeedContent.WithContext(r.ctx).
				Where(q.RanFeedContent.HotScore.Lt(cursorScore)).
				Or(q.RanFeedContent.HotScore.Eq(cursorScore), q.RanFeedContent.ID.Lt(cursorID)),
		)
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
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
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

// ListPublishedByAuthorWithinWindow 取作者在 sinceMillis 之后发布的 PUBLIC 内容 按 published_at 倒序 capped limit
func (r *ContentRepositoryImpl) ListPublishedByAuthorWithinWindow(authorID int64, sinceMillis int64, limit int) ([]*model.RanFeedContent, error) {
	if authorID <= 0 || limit <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	doQuery := q.RanFeedContent.WithContext(r.ctx).
		Select(q.RanFeedContent.ID, q.RanFeedContent.ContentType, q.RanFeedContent.UserID, q.RanFeedContent.PublishedAt).
		Where(q.RanFeedContent.UserID.Eq(authorID)).
		Where(q.RanFeedContent.Status.Eq(contentEnum.ContentStatusPublished.Int32())).
		Where(q.RanFeedContent.Visibility.Eq(contentEnum.VisibilityPublic.Int32())).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
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
			q.RanFeedContent.PublishedAt,
		).
		Where(q.RanFeedContent.Status.Eq(status)).
		Where(q.RanFeedContent.Visibility.Eq(visibility)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
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
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
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
		Where(q.RanFeedContent.Status.Eq(contentEnum.ContentStatusPublished.Int32())).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
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
		Where(q.RanFeedContent.Status.Eq(contentEnum.ContentStatusPublished.Int32())).
		Where(q.RanFeedContent.Visibility.Eq(contentEnum.VisibilityPublic.Int32())).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
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
		Where(q.RanFeedContent.Status.Eq(contentEnum.ContentStatusPublished.Int32())).
		Where(q.RanFeedContent.Visibility.Eq(contentEnum.VisibilityPublic.Int32())).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Where(q.RanFeedContent.PublishedAt.IsNotNull())

	if cursorID > 0 {
		doQuery = doQuery.Where(q.RanFeedContent.ID.Gt(cursorID))
	}

	return doQuery.Order(q.RanFeedContent.ID).Limit(limit).Find()
}

// BatchUpdateHotScores 批量更新热度分与更新时间
// 每批一条 CASE WHEN 语句 替代事务内逐条 UPDATE 缩短热榜任务耗时
func (r *ContentRepositoryImpl) BatchUpdateHotScores(ids []int64, scores []float64, updatedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) != len(scores) {
		return fmt.Errorf("ids and scores length mismatch")
	}
	for start := 0; start < len(ids); start += hotScoreBatchSize {
		end := start + hotScoreBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		sql, args := buildHotScoreUpdateSQL(ids[start:end], scores[start:end], updatedAt)
		if err := r.db.DB.WithContext(r.ctx).Exec(sql, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

// buildHotScoreUpdateSQL 构造 CASE WHEN 批量更新语句 只触碰未删除行
func buildHotScoreUpdateSQL(ids []int64, scores []float64, updatedAt time.Time) (string, []any) {
	var sb strings.Builder
	sb.WriteString("UPDATE ran_feed_content SET hot_score = CASE id")
	args := make([]any, 0, len(ids)*3+1)
	for i, id := range ids {
		sb.WriteString(" WHEN ? THEN ?")
		args = append(args, id, scores[i])
	}
	sb.WriteString(" END, last_hot_score_at = ?")
	args = append(args, updatedAt)

	sb.WriteString(" WHERE is_deleted = 0 AND id IN (")
	for i, id := range ids {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString("?")
		args = append(args, id)
	}
	sb.WriteString(")")
	return sb.String(), args
}

// AdminPageContents 管理端分页查询
func (r *ContentRepositoryImpl) AdminPageContents(status *int32, contentType *int32, authorID *int64, offset, limit int) ([]*model.RanFeedContent, int64, error) {
	if limit <= 0 {
		return []*model.RanFeedContent{}, 0, nil
	}
	q := r.getQuery()
	stmt := q.RanFeedContent.WithContext(r.ctx).Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32()))
	if status != nil {
		stmt = stmt.Where(q.RanFeedContent.Status.Eq(*status))
	}
	if contentType != nil {
		stmt = stmt.Where(q.RanFeedContent.ContentType.Eq(*contentType))
	}
	if authorID != nil {
		stmt = stmt.Where(q.RanFeedContent.UserID.Eq(*authorID))
	}
	return stmt.
		Select(
			q.RanFeedContent.ID,
			q.RanFeedContent.UserID,
			q.RanFeedContent.ContentType,
			q.RanFeedContent.Status,
			q.RanFeedContent.Visibility,
			q.RanFeedContent.PublishedAt,
			q.RanFeedContent.CreatedAt,
		).
		Order(q.RanFeedContent.ID.Desc()).
		FindByPage(offset, limit)
}

// AdminGetByID 管理端取任意状态内容
func (r *ContentRepositoryImpl) AdminGetByID(contentID int64) (*model.RanFeedContent, error) {
	if contentID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(r.ctx).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// AdminUpdateStatus 管理端条件翻转状态
func (r *ContentRepositoryImpl) AdminUpdateStatus(contentID int64, fromStatus, toStatus int32, operatorID int64) (int64, error) {

	q := r.getQuery()
	info, err := q.RanFeedContent.WithContext(r.ctx).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Where(q.RanFeedContent.Status.Eq(fromStatus)).
		UpdateSimple(
			q.RanFeedContent.Status.Value(toStatus),
			q.RanFeedContent.UpdatedBy.Value(operatorID),
		)
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}

// GetOwnedByID 取本人任意状态未删内容
func (r *ContentRepositoryImpl) GetOwnedByID(contentID, userID int64) (*model.RanFeedContent, error) {
	if contentID <= 0 || userID <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(r.ctx).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Where(q.RanFeedContent.UserID.Eq(userID)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Take()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// UpdateDraftMeta 草稿编辑更新内容级字段 状态回落 DRAFT 被拒内容编辑后重新变草稿
func (r *ContentRepositoryImpl) UpdateDraftMeta(contentID int64, visibility int32, updatedBy int64) error {
	if contentID <= 0 {
		return nil
	}
	q := r.getQuery()
	_, err := q.RanFeedContent.WithContext(r.ctx).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		UpdateSimple(
			q.RanFeedContent.Status.Value(contentEnum.ContentStatusDraft.Int32()),
			q.RanFeedContent.Visibility.Value(visibility),
			q.RanFeedContent.UpdatedBy.Value(updatedBy),
		)
	return err
}

// SubmitOwned 本人 fromStatuses 内内容提交进审核 条件更新防并发 返回受影响行数
func (r *ContentRepositoryImpl) SubmitOwned(contentID, userID int64, fromStatuses []int32, toStatus int32, updatedBy int64) (int64, error) {
	if contentID <= 0 || userID <= 0 || len(fromStatuses) == 0 {
		return 0, nil
	}
	q := r.getQuery()
	info, err := q.RanFeedContent.WithContext(r.ctx).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Where(q.RanFeedContent.UserID.Eq(userID)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Where(q.RanFeedContent.Status.In(fromStatuses...)).
		UpdateSimple(
			q.RanFeedContent.Status.Value(toStatus),
			q.RanFeedContent.UpdatedBy.Value(updatedBy),
		)
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}

// MyContentPage 我的内容全状态列表 keyset 游标 id 倒序 cursorID 为正取更早的
func (r *ContentRepositoryImpl) MyContentPage(userID int64, status *int32, contentType *int32, cursorID int64, limit int) ([]*model.RanFeedContent, error) {
	if userID <= 0 || limit <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	stmt := q.RanFeedContent.WithContext(r.ctx).
		Select(
			q.RanFeedContent.ID,
			q.RanFeedContent.ContentType,
			q.RanFeedContent.Status,
			q.RanFeedContent.Visibility,
			q.RanFeedContent.PublishedAt,
			q.RanFeedContent.CreatedAt,
		).
		Where(q.RanFeedContent.UserID.Eq(userID)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32()))
	if status != nil {
		stmt = stmt.Where(q.RanFeedContent.Status.Eq(*status))
	}
	if contentType != nil {
		stmt = stmt.Where(q.RanFeedContent.ContentType.Eq(*contentType))
	}
	if cursorID > 0 {
		stmt = stmt.Where(q.RanFeedContent.ID.Lt(cursorID))
	}
	// 游标即 id 排序也按 id 倒序 与游标口径一致 雪花 id 天然近似创建时间序
	return stmt.
		Order(q.RanFeedContent.ID.Desc()).
		Limit(limit).
		Find()
}

// AdminApproveContent 审核通过 仅对 PENDING_REVIEW 生效 落 published_at + updated_by 返回受影响行数
func (r *ContentRepositoryImpl) AdminApproveContent(contentID, operatorID int64, publishedAt time.Time) (int64, error) {
	if contentID <= 0 {
		return 0, nil
	}

	q := r.getQuery()
	info, err := q.RanFeedContent.WithContext(r.ctx).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Where(q.RanFeedContent.Status.Eq(contentEnum.ContentStatusPendingReview.Int32())).
		UpdateSimple(
			q.RanFeedContent.Status.Value(contentEnum.ContentStatusPublished.Int32()),
			q.RanFeedContent.PublishedAt.Value(publishedAt),
			q.RanFeedContent.UpdatedBy.Value(operatorID),
		)
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}
