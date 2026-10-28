// Package memberrepo 会话成员与阅读游标读写
package memberrepo

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/pkg/enums"
)

var _ repositories.MemberRepository = (*Repository)(nil)

// Repository 会话成员仓储
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
func (r *Repository) WithTx(tx *query.Query) repositories.MemberRepository {
	return &Repository{tx: tx}
}

// Upsert 幂等写成员
func (r *Repository) Upsert(ctx context.Context, rows []*model.RanFeedImConversationMember) error {
	if len(rows) == 0 {
		return nil
	}
	return r.getQuery().RanFeedImConversationMember.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "conversation_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"is_deleted", "updated_by", "updated_at"}),
		}).
		Create(rows...)
}

// UpsertJoin 入群写成员,冲突时同步可见起点与已读水位
func (r *Repository) UpsertJoin(ctx context.Context, rows []*model.RanFeedImConversationMember) error {
	if len(rows) == 0 {
		return nil
	}
	return r.getQuery().RanFeedImConversationMember.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "conversation_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"is_deleted", "min_seq", "read_seq", "muted", "updated_by", "updated_at"}),
		}).
		Create(rows...)
}

// Get 查单个成员
func (r *Repository) Get(ctx context.Context, conversationID, userID int64) (*model.RanFeedImConversationMember, error) {
	if conversationID <= 0 || userID <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedImConversationMember.WithContext(ctx).
		Where(q.RanFeedImConversationMember.ConversationID.Eq(conversationID)).
		Where(q.RanFeedImConversationMember.UserID.Eq(userID)).
		Where(q.RanFeedImConversationMember.IsDeleted.Eq(enums.NotDeleted.Int32())).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// BatchGetByUser 批量取某用户的成员记录
func (r *Repository) BatchGetByUser(ctx context.Context, userID int64, conversationIDs []int64) (map[int64]*model.RanFeedImConversationMember, error) {
	out := make(map[int64]*model.RanFeedImConversationMember, len(conversationIDs))
	if userID <= 0 || len(conversationIDs) == 0 {
		return out, nil
	}
	q := r.getQuery()
	rows, err := q.RanFeedImConversationMember.WithContext(ctx).
		Where(q.RanFeedImConversationMember.UserID.Eq(userID)).
		Where(q.RanFeedImConversationMember.ConversationID.In(conversationIDs...)).
		Where(q.RanFeedImConversationMember.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ConversationID] = row
	}
	return out, nil
}

// ListByConversation 取会话全部活跃成员
func (r *Repository) ListByConversation(ctx context.Context, conversationID int64) ([]*model.RanFeedImConversationMember, error) {
	if conversationID <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	return q.RanFeedImConversationMember.WithContext(ctx).
		Where(q.RanFeedImConversationMember.ConversationID.Eq(conversationID)).
		Where(q.RanFeedImConversationMember.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Order(q.RanFeedImConversationMember.CreatedAt.Asc(), q.RanFeedImConversationMember.ID.Asc()).
		Find()
}

// AdvanceReadSeq 只增不减推进已读水位
func (r *Repository) AdvanceReadSeq(ctx context.Context, conversationID, userID, readSeq int64) (int64, error) {
	if conversationID <= 0 || userID <= 0 || readSeq <= 0 {
		return 0, nil
	}
	q := r.getQuery()
	info, err := q.RanFeedImConversationMember.WithContext(ctx).
		Where(q.RanFeedImConversationMember.ConversationID.Eq(conversationID)).
		Where(q.RanFeedImConversationMember.UserID.Eq(userID)).
		Where(q.RanFeedImConversationMember.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Where(q.RanFeedImConversationMember.ReadSeq.Lt(readSeq)).
		UpdateSimple(
			q.RanFeedImConversationMember.ReadSeq.Value(readSeq),
			q.RanFeedImConversationMember.UpdatedBy.Value(userID),
		)
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}

// MarkLeft 标记成员退群
func (r *Repository) MarkLeft(ctx context.Context, conversationID, userID int64) (int64, error) {
	if conversationID <= 0 || userID <= 0 {
		return 0, nil
	}
	q := r.getQuery()
	info, err := q.RanFeedImConversationMember.WithContext(ctx).
		Where(q.RanFeedImConversationMember.ConversationID.Eq(conversationID)).
		Where(q.RanFeedImConversationMember.UserID.Eq(userID)).
		Where(q.RanFeedImConversationMember.IsDeleted.Eq(enums.NotDeleted.Int32())).
		UpdateSimple(
			q.RanFeedImConversationMember.IsDeleted.Value(enums.Deleted.Int32()),
			q.RanFeedImConversationMember.UpdatedBy.Value(userID),
		)
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}
