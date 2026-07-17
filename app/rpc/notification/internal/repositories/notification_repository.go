package repositories

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/notification/internal/entity/model"
	"ran-feed/app/rpc/notification/internal/entity/query"
	"ran-feed/pkg/orm"
)

// NotificationRepository 通知域数据层
// 读加 IsDeleted.Eq(0) 软删过滤 遵守 CLAUDE.md 不变式
type NotificationRepository interface {
	WithTx(tx *query.Query) NotificationRepository
	// UpsertAggregate 聚合 upsert LIKE_FAVORITE 与 FOLLOW 走该路径
	// ON DUPLICATE KEY(uk_recipient_aggkey) 命中则 agg_count+1 & actor_id 换为最新 & re-surface 变未读
	UpsertAggregate(row *model.RanFeedNotification) error
	// Insert 单条 insert COMMENT_REPLY 走该路径 agg_key 天然唯一(CR:{comment_id})
	Insert(row *model.RanFeedNotification) error
	// ListByRecipient 收件箱复合游标查询 按(updated_at DESC id DESC)排序 一次多取 1 条供 logic 层判 has_more
	// cursorUpdatedAt 零值视为首页 typeFilter 为 0 视为不限
	ListByRecipient(recipientID int64, typeFilter int32, cursorUpdatedAt time.Time, cursorID int64, limit int) ([]*model.RanFeedNotification, error)
	// CountUnread 未读数 走 idx_recipient_unread
	CountUnread(recipientID int64) (int64, error)
	// MarkRead 按 ids 标已读 recipient 入 where 防越权 只翻未读避免重复计数 返回真实变更行数
	MarkRead(recipientID int64, ids []int64) (int64, error)
	// MarkAllRead 全部标已读 recipient 入 where 只翻未读 返回变更行数
	MarkAllRead(recipientID int64) (int64, error)
}

type notificationRepositoryImpl struct {
	ctx context.Context
	db  *orm.DB
	logx.Logger
	tx *query.Query
}

func NewNotificationRepository(ctx context.Context, db *orm.DB) NotificationRepository {
	return &notificationRepositoryImpl{
		ctx:    ctx,
		db:     db,
		Logger: logx.WithContext(ctx),
	}
}

func (r *notificationRepositoryImpl) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

func (r *notificationRepositoryImpl) WithTx(tx *query.Query) NotificationRepository {
	return &notificationRepositoryImpl{
		ctx:    r.ctx,
		db:     r.db,
		tx:     tx,
		Logger: r.Logger,
	}
}

func (r *notificationRepositoryImpl) UpsertAggregate(row *model.RanFeedNotification) error {
	if row == nil || row.RecipientID <= 0 || row.AggKey == "" {
		return nil
	}
	if row.AggCount <= 0 {
		row.AggCount = 1
	}

	q := r.getQuery()
	db := q.RanFeedNotification.WithContext(r.ctx).UnderlyingDB()
	// 命中 uk_recipient_aggkey 时:
	//   agg_count+1 累加互动次数
	//   actor_id 换为最新触发者
	//   re-surface 冒到顶(updated_at 由 ON UPDATE CURRENT_TIMESTAMP(3) 自动刷)
	//   is_read=0 让老通知重新变未读 保证再触达
	return db.Exec(
		`INSERT INTO ran_feed_notification
			(recipient_id, actor_id, notify_type, agg_key, agg_count,
			 content_id, comment_id, snippet, is_read, version,
			 created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 1, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			agg_count = agg_count + 1,
			actor_id = VALUES(actor_id),
			is_read = 0,
			updated_at = VALUES(updated_at),
			version = version + 1`,
		row.RecipientID,
		row.ActorID,
		row.NotifyType,
		row.AggKey,
		row.AggCount,
		row.ContentID,
		row.CommentID,
		row.Snippet,
		row.CreatedBy,
		row.UpdatedBy,
		row.CreatedAt,
		row.UpdatedAt,
	).Error
}

func (r *notificationRepositoryImpl) Insert(row *model.RanFeedNotification) error {
	if row == nil || row.RecipientID <= 0 || row.AggKey == "" {
		return nil
	}
	if row.AggCount <= 0 {
		row.AggCount = 1
	}

	q := r.getQuery()
	return q.RanFeedNotification.WithContext(r.ctx).Create(row)
}

func (r *notificationRepositoryImpl) ListByRecipient(recipientID int64, typeFilter int32, cursorUpdatedAt time.Time, cursorID int64, limit int) ([]*model.RanFeedNotification, error) {
	if recipientID <= 0 || limit <= 0 {
		return nil, nil
	}

	q := r.getQuery()
	doQuery := q.RanFeedNotification.WithContext(r.ctx).
		Where(q.RanFeedNotification.RecipientID.Eq(recipientID)).
		Where(q.RanFeedNotification.IsDeleted.Eq(0))
	if typeFilter > 0 {
		doQuery = doQuery.Where(q.RanFeedNotification.NotifyType.Eq(typeFilter))
	}
	// 复合游标 updated_at DESC id DESC over-fetch+1 由 logic 层切片判 has_more
	// 用子 DO 分组 保证 OR 被括号包住不破坏外层 recipient/软删/type 过滤
	if !cursorUpdatedAt.IsZero() {
		doQuery = doQuery.Where(
			q.RanFeedNotification.WithContext(r.ctx).
				Where(q.RanFeedNotification.UpdatedAt.Lt(cursorUpdatedAt)).
				Or(q.RanFeedNotification.UpdatedAt.Eq(cursorUpdatedAt), q.RanFeedNotification.ID.Lt(cursorID)),
		)
	}

	rows, err := doQuery.
		Order(q.RanFeedNotification.UpdatedAt.Desc(), q.RanFeedNotification.ID.Desc()).
		Limit(limit).
		Find()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *notificationRepositoryImpl) CountUnread(recipientID int64) (int64, error) {
	if recipientID <= 0 {
		return 0, nil
	}

	q := r.getQuery()
	return q.RanFeedNotification.WithContext(r.ctx).
		Where(q.RanFeedNotification.RecipientID.Eq(recipientID)).
		Where(q.RanFeedNotification.IsDeleted.Eq(0)).
		Where(q.RanFeedNotification.IsRead.Eq(0)).
		Count()
}

func (r *notificationRepositoryImpl) MarkRead(recipientID int64, ids []int64) (int64, error) {
	if recipientID <= 0 || len(ids) == 0 {
		return 0, nil
	}

	q := r.getQuery()
	now := time.Now()
	info, err := q.RanFeedNotification.WithContext(r.ctx).
		Where(q.RanFeedNotification.RecipientID.Eq(recipientID)).
		Where(q.RanFeedNotification.ID.In(ids...)).
		Where(q.RanFeedNotification.IsDeleted.Eq(0)).
		Where(q.RanFeedNotification.IsRead.Eq(0)).
		Updates(map[string]interface{}{
			"is_read":    1,
			"read_at":    now,
			"updated_at": now,
			"updated_by": recipientID,
		})
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}

func (r *notificationRepositoryImpl) MarkAllRead(recipientID int64) (int64, error) {
	if recipientID <= 0 {
		return 0, nil
	}

	q := r.getQuery()
	now := time.Now()
	info, err := q.RanFeedNotification.WithContext(r.ctx).
		Where(q.RanFeedNotification.RecipientID.Eq(recipientID)).
		Where(q.RanFeedNotification.IsDeleted.Eq(0)).
		Where(q.RanFeedNotification.IsRead.Eq(0)).
		Updates(map[string]interface{}{
			"is_read":    1,
			"read_at":    now,
			"updated_at": now,
			"updated_by": recipientID,
		})
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}