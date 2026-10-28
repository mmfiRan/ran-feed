// Package messagerepo 消息表读写
package messagerepo

import (
	"context"
	"errors"

	"gorm.io/gorm"

	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/pkg/enums"
)

var _ repositories.MessageRepository = (*Repository)(nil)

// Repository 消息仓储
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
func (r *Repository) WithTx(tx *query.Query) repositories.MessageRepository {
	return &Repository{tx: tx}
}

// Create 新建消息
func (r *Repository) Create(ctx context.Context, row *model.RanFeedImMessage) error {
	if row == nil {
		return nil
	}
	return r.getQuery().RanFeedImMessage.WithContext(ctx).Create(row)
}

// GetByClientMsgID 按客户端消息标识查
func (r *Repository) GetByClientMsgID(ctx context.Context, conversationID int64, clientMsgID string) (*model.RanFeedImMessage, error) {
	if conversationID <= 0 || clientMsgID == "" {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedImMessage.WithContext(ctx).
		Where(q.RanFeedImMessage.ConversationID.Eq(conversationID)).
		Where(q.RanFeedImMessage.ClientMsgID.Eq(clientMsgID)).
		Where(q.RanFeedImMessage.IsDeleted.Eq(enums.NotDeleted.Int32())).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// GetBySeq 按会话内序号查
func (r *Repository) GetBySeq(ctx context.Context, conversationID, seq int64) (*model.RanFeedImMessage, error) {
	if conversationID <= 0 || seq <= 0 {
		return nil, nil
	}
	q := r.getQuery()
	row, err := q.RanFeedImMessage.WithContext(ctx).
		Where(q.RanFeedImMessage.ConversationID.Eq(conversationID)).
		Where(q.RanFeedImMessage.Seq.Eq(seq)).
		Where(q.RanFeedImMessage.IsDeleted.Eq(enums.NotDeleted.Int32())).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
}

// ListBySeqRange 拉取序号区间消息
func (r *Repository) ListBySeqRange(ctx context.Context, conversationID, seqBegin, seqEnd int64) ([]*model.RanFeedImMessage, error) {
	if conversationID <= 0 || seqEnd <= seqBegin {
		return nil, nil
	}
	q := r.getQuery()
	return q.RanFeedImMessage.WithContext(ctx).
		Where(q.RanFeedImMessage.ConversationID.Eq(conversationID)).
		Where(q.RanFeedImMessage.Seq.Gt(seqBegin)).
		Where(q.RanFeedImMessage.Seq.Lte(seqEnd)).
		Where(q.RanFeedImMessage.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Order(q.RanFeedImMessage.Seq.Asc()).
		Find()
}

// BatchGetByIDs 批量取消息
func (r *Repository) BatchGetByIDs(ctx context.Context, messageIDs []int64) (map[int64]*model.RanFeedImMessage, error) {
	out := make(map[int64]*model.RanFeedImMessage, len(messageIDs))
	if len(messageIDs) == 0 {
		return out, nil
	}
	q := r.getQuery()
	rows, err := q.RanFeedImMessage.WithContext(ctx).
		Where(q.RanFeedImMessage.ID.In(messageIDs...)).
		Where(q.RanFeedImMessage.IsDeleted.Eq(enums.NotDeleted.Int32())).
		Find()
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

// Revoke 撤回消息并清空正文
func (r *Repository) Revoke(ctx context.Context, conversationID, seq, senderID int64) (int64, error) {
	if conversationID <= 0 || seq <= 0 || senderID <= 0 {
		return 0, nil
	}
	q := r.getQuery()
	info, err := q.RanFeedImMessage.WithContext(ctx).
		Where(q.RanFeedImMessage.ConversationID.Eq(conversationID)).
		Where(q.RanFeedImMessage.Seq.Eq(seq)).
		Where(q.RanFeedImMessage.SenderID.Eq(senderID)).
		Where(q.RanFeedImMessage.Status.Eq(imEnum.MessageStatusNormal.Int32())).
		Where(q.RanFeedImMessage.IsDeleted.Eq(enums.NotDeleted.Int32())).
		UpdateSimple(
			q.RanFeedImMessage.Status.Value(imEnum.MessageStatusRevoked.Int32()),
			q.RanFeedImMessage.Content.Value(""),
			q.RanFeedImMessage.UpdatedBy.Value(senderID),
		)
	if err != nil {
		return 0, err
	}
	return info.RowsAffected, nil
}
