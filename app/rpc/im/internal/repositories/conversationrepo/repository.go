// Package conversationrepo 会话主表读写
package conversationrepo

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/pkg/enums"
)

var _ repositories.ConversationRepository = (*Repository)(nil)

// Repository 会话主表仓储
type Repository struct {
	tx *query.Query
}

// New 构造仓储
func New() *Repository {
	return &Repository{}
}

func (r *Repository) getQuery() *query.Query {
	if r.tx != nil {
		return r.tx
	}
	return query.Q
}

// WithTx 绑定事务
func (r *Repository) WithTx(tx *query.Query) repositories.ConversationRepository {
	return &Repository{tx: tx}
}

// Create 新建会话
func (r *Repository) Create(ctx context.Context, row *model.RanFeedImConversation) error {
	if row == nil {
		return nil
	}
	return r.getQuery().RanFeedImConversation.WithContext(ctx).Create(row)
}

// GetByID 按 ID 查会话
func (r *Repository) GetByID(ctx context.Context, conversationID int64) (*model.RanFeedImConversation, error) {
	if conversationID <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedImConversation.WithContext(ctx).
		Where(q.RanFeedImConversation.ID.Eq(conversationID)).
		Where(q.RanFeedImConversation.IsDeleted.Eq(enums.NotDeleted.Int32())).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// GetByIDForUpdate 行锁查会话,seq 分配与成员数变更在同一行锁上串行
func (r *Repository) GetByIDForUpdate(ctx context.Context, conversationID int64) (*model.RanFeedImConversation, error) {
	if conversationID <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedImConversation.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where(q.RanFeedImConversation.ID.Eq(conversationID)).
		Where(q.RanFeedImConversation.IsDeleted.Eq(enums.NotDeleted.Int32())).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// GetByBizKey 按业务幂等键查会话
func (r *Repository) GetByBizKey(ctx context.Context, bizKey string) (*model.RanFeedImConversation, error) {
	if bizKey == "" {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedImConversation.WithContext(ctx).
		Where(q.RanFeedImConversation.BizKey.Eq(bizKey)).
		Where(q.RanFeedImConversation.IsDeleted.Eq(enums.NotDeleted.Int32())).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// UpdateSeqWatermark 推进 max_seq 并刷新最后消息字段
func (r *Repository) UpdateSeqWatermark(ctx context.Context, conversationID, seq, lastMsgID, senderID int64, lastMsgAt time.Time) error {
	if conversationID <= 0 || seq <= 0 {
		return nil
	}
	q := r.getQuery()
	_, err := q.RanFeedImConversation.WithContext(ctx).
		Where(q.RanFeedImConversation.ID.Eq(conversationID)).
		Where(q.RanFeedImConversation.IsDeleted.Eq(enums.NotDeleted.Int32())).
		UpdateSimple(
			q.RanFeedImConversation.MaxSeq.Value(seq),
			q.RanFeedImConversation.LastMsgID.Value(lastMsgID),
			q.RanFeedImConversation.LastMsgAt.Value(lastMsgAt),
			q.RanFeedImConversation.UpdatedBy.Value(senderID),
		)
	return err
}

// BatchGetByIDs 批量取会话
func (r *Repository) BatchGetByIDs(ctx context.Context, conversationIDs []int64) (map[int64]*model.RanFeedImConversation, error) {
	out := make(map[int64]*model.RanFeedImConversation, len(conversationIDs))
	if len(conversationIDs) == 0 {
		return out, nil
	}
	q := r.getQuery()
	rows, err := q.RanFeedImConversation.WithContext(ctx).
		Where(q.RanFeedImConversation.ID.In(conversationIDs...)).
		Where(q.RanFeedImConversation.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

// ListIDsByUser 用户会话 ID 列表
func (r *Repository) ListIDsByUser(ctx context.Context, userID int64, cursorLastMsgAt time.Time, cursorID int64, limit int) ([]int64, error) {
	if userID <= 0 || limit <= 0 {
		return nil, nil
	}
	// 两张表都有 id,用表别名显式限定避免 Column 'id' is ambiguous
	db := r.getQuery().RanFeedImConversation.WithContext(ctx).UnderlyingDB()
	stmt := db.Table(model.TableNameRanFeedImConversation+" AS c").
		Select("c.id").
		Joins("JOIN "+model.TableNameRanFeedImConversationMember+" AS m ON m.conversation_id = c.id").
		Where("m.user_id = ?", userID).
		Where("m.is_deleted = ?", enums.NotDeleted.Int32()).
		Where("c.is_deleted = ?", enums.NotDeleted.Int32())
	if !cursorLastMsgAt.IsZero() {
		stmt = stmt.Where("(c.last_msg_at < ? OR (c.last_msg_at = ? AND c.id < ?))",
			cursorLastMsgAt, cursorLastMsgAt, cursorID)
	}
	var ids []int64
	if err := stmt.
		Order("c.last_msg_at DESC, c.id DESC").
		Limit(limit).
		Pluck("c.id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}
