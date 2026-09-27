// 本文件负责内容主表的读写 发布 下架 热榜候选 关注流与推荐流分页查询都经由此处
// 只做数据读写 状态判定与可见性过滤由调用方以参数给出
package contentrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ran-feed/app/rpc/content/internal/common/consts"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/pkg/enums"

	"gorm.io/gorm"
)

var _ repositories.ContentRepository = (*Repository)(nil)

type Repository struct {
	tx *query.Query
}

func New() *Repository {
	return &Repository{}
}

func (r *Repository) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *Repository) WithTx(tx *query.Query) repositories.ContentRepository {
	return &Repository{
		tx: tx,
	}
}

func (r *Repository) GetDetailByID(ctx context.Context, contentID int64) (*model.RanFeedContent, error) {
	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) GetByIDBrief(ctx context.Context, contentID int64) (*model.RanFeedContent, error) {
	if contentID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) GetHotScoreByID(ctx context.Context, contentID int64) (float64, error) {
	if contentID <= 0 {
		return 0, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(ctx).
		Select(q.RanFeedContent.HotScore).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Take()
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 0, fmt.Errorf("content not found: content_id=%d", contentID)
	}
	return row.HotScore, nil
}

func (r *Repository) CountByAuthor(ctx context.Context, status int32, visibility int32, authorID int64) (int64, error) {
	if authorID <= 0 {
		return 0, nil
	}
	q := r.getQuery()
	return q.RanFeedContent.WithContext(ctx).
		Where(q.RanFeedContent.UserID.Eq(authorID)).
		Where(q.RanFeedContent.Status.Eq(status)).
		Where(q.RanFeedContent.Visibility.Eq(visibility)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Where(q.RanFeedContent.PublishedAt.IsNotNull()).
		Count()
}

func (r *Repository) ListRecommendByHotScoreCursor(ctx context.Context, status int32, visibility int32, cursorScore float64, cursorID int64, limit int) ([]*model.RanFeedContent, error) {
	if limit <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	doQuery := q.RanFeedContent.WithContext(ctx).
		Select(q.RanFeedContent.ID, q.RanFeedContent.ContentType, q.RanFeedContent.UserID, q.RanFeedContent.PublishedAt, q.RanFeedContent.HotScore).
		Where(q.RanFeedContent.Status.Eq(status)).
		Where(q.RanFeedContent.Visibility.Eq(visibility)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Where(q.RanFeedContent.PublishedAt.IsNotNull())

	if cursorID > 0 {
		doQuery = doQuery.Where(
			q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) ListFollowByAuthorsCursor(ctx context.Context, status int32, visibility int32, authorIDs []int64, cursorMillis int64, limit int) ([]*model.RanFeedContent, error) {
	if limit <= 0 {
		return nil, nil
	}
	if len(authorIDs) == 0 {
		return nil, nil
	}

	q := r.getQuery()
	doQuery := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) ListPublishedByAuthorWithinWindow(ctx context.Context, authorID int64, sinceMillis int64, limit int) ([]*model.RanFeedContent, error) {
	if authorID <= 0 || limit <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	doQuery := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) ListColdUpdateContents(ctx context.Context, status int32, visibility int32, start time.Time, cursorID int64, limit int) ([]*model.RanFeedContent, error) {
	if limit <= 0 {
		return nil, nil
	}

	q := r.getQuery()

	doQuery := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) ScanIndexableByIDCursor(ctx context.Context, cursorID int64, limit int) ([]*model.RanFeedContent, error) {
	if limit <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	doQuery := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) GetOwnedByID(ctx context.Context, contentID, userID int64) (*model.RanFeedContent, error) {
	if contentID <= 0 || userID <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) MyContentPage(ctx context.Context, userID int64, status *int32, contentType *int32, cursorID int64, limit int) ([]*model.RanFeedContent, error) {
	if userID <= 0 || limit <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	stmt := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) AdminGetByID(ctx context.Context, contentID int64) (*model.RanFeedContent, error) {
	if contentID <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	row, err := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) AdminPageContents(ctx context.Context, status *int32, contentType *int32, authorID *int64, offset, limit int) ([]*model.RanFeedContent, int64, error) {
	if limit <= 0 {
		return []*model.RanFeedContent{}, 0, nil
	}
	q := r.getQuery()
	stmt := q.RanFeedContent.WithContext(ctx).Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32()))
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

func (r *Repository) CreateContent(ctx context.Context, content *model.RanFeedContent) error {
	q := r.getQuery()
	return q.RanFeedContent.WithContext(ctx).Create(content)
}

func (r *Repository) DeleteByID(ctx context.Context, contentID int64) error {
	q := r.getQuery()
	_, err := q.RanFeedContent.WithContext(ctx).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		UpdateSimple(q.RanFeedContent.IsDeleted.Value(enums.Deleted.Int32()))
	return err
}

func (r *Repository) AdminUpdateStatus(ctx context.Context, contentID int64, fromStatus, toStatus int32, operatorID int64) (int64, error) {

	q := r.getQuery()
	info, err := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) AdminApproveContent(ctx context.Context, contentID, operatorID int64, publishedAt time.Time) (int64, error) {
	if contentID <= 0 {
		return 0, nil
	}

	q := r.getQuery()
	info, err := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) UpdateDraftMeta(ctx context.Context, contentID int64, visibility int32, updatedBy int64) error {
	if contentID <= 0 {
		return nil
	}
	q := r.getQuery()
	_, err := q.RanFeedContent.WithContext(ctx).
		Where(q.RanFeedContent.ID.Eq(contentID)).
		Where(q.RanFeedContent.IsDeleted.Eq(enums.NotDeleted.Int32())).
		UpdateSimple(
			q.RanFeedContent.Status.Value(contentEnum.ContentStatusDraft.Int32()),
			q.RanFeedContent.Visibility.Value(visibility),
			q.RanFeedContent.UpdatedBy.Value(updatedBy),
		)
	return err
}

func (r *Repository) SubmitOwned(ctx context.Context, contentID, userID int64, fromStatuses []int32, toStatus int32, updatedBy int64) (int64, error) {
	if contentID <= 0 || userID <= 0 || len(fromStatuses) == 0 {
		return 0, nil
	}
	q := r.getQuery()
	info, err := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) BatchGetRecommendByIDs(ctx context.Context, status int32, visibility int32, contentIDs []int64) (map[int64]*model.RanFeedContent, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedContent{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) BatchGetPublishedByIDs(ctx context.Context, contentIDs []int64) (map[int64]*model.RanFeedContent, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedContent{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) BatchGetIndexableByIDs(ctx context.Context, contentIDs []int64) (map[int64]*model.RanFeedContent, error) {
	if len(contentIDs) == 0 {
		return map[int64]*model.RanFeedContent{}, nil
	}

	q := r.getQuery()
	rows, err := q.RanFeedContent.WithContext(ctx).
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

func (r *Repository) BatchUpdateHotScores(ctx context.Context, ids []int64, scores []float64, updatedAt time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) != len(scores) {
		return fmt.Errorf("ids and scores length mismatch")
	}
	content := r.getQuery().RanFeedContent
	for start := 0; start < len(ids); start += consts.HotScoreUpdateBatchSize {
		end := start + consts.HotScoreUpdateBatchSize
		if end > len(ids) {
			end = len(ids)
		}
		batchIDs := ids[start:end]
		batchScores := scores[start:end]

		var caseSQL strings.Builder
		caseSQL.WriteString("CASE id")
		caseArgs := make([]any, 0, len(batchIDs)*2)
		for i, id := range batchIDs {
			caseSQL.WriteString(" WHEN ? THEN ?")
			caseArgs = append(caseArgs, id, batchScores[i])
		}
		caseSQL.WriteString(" END")

		_, err := content.WithContext(ctx).
			Where(content.IsDeleted.Eq(enums.NotDeleted.Int32()), content.ID.In(batchIDs...)).
			Updates(map[string]any{
				"hot_score":         gorm.Expr(caseSQL.String(), caseArgs...),
				"last_hot_score_at": updatedAt,
			})
		if err != nil {
			return err
		}
	}
	return nil
}
