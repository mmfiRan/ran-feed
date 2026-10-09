package repositories

import (
	"context"
	"time"

	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/entity/query"
)

type ContentRepository interface {
	WithTx(tx *query.Query) ContentRepository
	CreateContent(ctx context.Context, content *model.RanFeedContent) error
	GetDetailByID(ctx context.Context, contentID int64) (*model.RanFeedContent, error)
	GetByIDBrief(ctx context.Context, contentID int64) (*model.RanFeedContent, error)
	DeleteByID(ctx context.Context, contentID int64) error
	GetHotScoreByID(ctx context.Context, contentID int64) (float64, error)
	CountByAuthor(ctx context.Context, status contentEnum.ContentStatusEnum, visibility contentEnum.VisibilityEnum, authorID int64) (int64, error)
	ListRecommendByHotScoreCursor(ctx context.Context, status contentEnum.ContentStatusEnum, visibility contentEnum.VisibilityEnum, cursorScore float64, cursorID int64, limit int) ([]*model.RanFeedContent, error)
	ListFollowByAuthorsCursor(ctx context.Context, status contentEnum.ContentStatusEnum, visibility contentEnum.VisibilityEnum, authorIDs []int64, cursorMillis int64, limit int) ([]*model.RanFeedContent, error)
	ListPublishedByAuthorWithinWindow(ctx context.Context, authorID int64, sinceMillis int64, limit int) ([]*model.RanFeedContent, error)
	ListColdUpdateContents(ctx context.Context, status contentEnum.ContentStatusEnum, visibility contentEnum.VisibilityEnum, start time.Time, cursorID int64, limit int) ([]*model.RanFeedContent, error)
	BatchGetRecommendByIDs(ctx context.Context, status contentEnum.ContentStatusEnum, visibility contentEnum.VisibilityEnum, contentIDs []int64) (map[int64]*model.RanFeedContent, error)
	BatchGetPublishedByIDs(ctx context.Context, contentIDs []int64) (map[int64]*model.RanFeedContent, error)
	BatchGetIndexableByIDs(ctx context.Context, contentIDs []int64) (map[int64]*model.RanFeedContent, error)
	ScanIndexableByIDCursor(ctx context.Context, cursorID int64, limit int) ([]*model.RanFeedContent, error)
	BatchUpdateHotScores(ctx context.Context, ids []int64, scores []float64, updatedAt time.Time) error
	// AdminPageContents 管理端多条件筛选
	AdminPageContents(ctx context.Context, status *contentEnum.ContentStatusEnum, contentType *contentEnum.ContentTypeEnum, authorID *int64, offset, limit int) ([]*model.RanFeedContent, int64, error)
	// AdminGetByID 管理端取任意状态内容
	AdminGetByID(ctx context.Context, contentID int64) (*model.RanFeedContent, error)
	// AdminUpdateStatus 管理端条件翻转状态
	AdminUpdateStatus(ctx context.Context, contentID int64, fromStatus, toStatus contentEnum.ContentStatusEnum, operatorID int64) (int64, error)
	// AdminApproveContent 审核通过 PENDING_REVIEW->PUBLISHED
	AdminApproveContent(ctx context.Context, contentID, operatorID int64, publishedAt time.Time) (int64, error)
	// GetOwnedByID 取本人任意状态内容 草稿编辑与提交鉴权用
	GetOwnedByID(ctx context.Context, contentID, userID int64) (*model.RanFeedContent, error)
	// UpdateDraftMeta 草稿编辑更新内容级字段 状态回落 DRAFT
	UpdateDraftMeta(ctx context.Context, contentID int64, visibility contentEnum.VisibilityEnum, updatedBy int64) error
	// SubmitOwned 本人指定状态提交进审核 条件更新返回受影响行数
	SubmitOwned(ctx context.Context, contentID, userID int64, fromStatuses []contentEnum.ContentStatusEnum, toStatus contentEnum.ContentStatusEnum, updatedBy int64) (int64, error)
	// MyContentPage 我的内容 全状态 keyset 游标按 id 倒序 id 与 created_at 同序
	MyContentPage(ctx context.Context, userID int64, status *contentEnum.ContentStatusEnum, contentType *contentEnum.ContentTypeEnum, cursorID int64, limit int) ([]*model.RanFeedContent, error)
}
