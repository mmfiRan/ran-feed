package repositories

import (
	"context"

	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
)

// MessageRepository 消息表数据层
type MessageRepository interface {
	WithTx(tx *query.Query) MessageRepository
	// Create 新建消息,唯一键冲突返回 gorm.ErrDuplicatedKey
	Create(ctx context.Context, row *model.RanFeedImMessage) error
	// GetByClientMsgID 查不到返回 (nil, nil)
	GetByClientMsgID(ctx context.Context, conversationID int64, clientMsgID string) (*model.RanFeedImMessage, error)
	// GetBySeq 查不到返回 (nil, nil)
	GetBySeq(ctx context.Context, conversationID, seq int64) (*model.RanFeedImMessage, error)
	// ListBySeqRange 拉取 (seqBegin, seqEnd] 区间消息,按 seq 升序
	ListBySeqRange(ctx context.Context, conversationID, seqBegin, seqEnd int64) ([]*model.RanFeedImMessage, error)
	// BatchGetByIDs 按消息 ID 批量取
	BatchGetByIDs(ctx context.Context, messageIDs []int64) (map[int64]*model.RanFeedImMessage, error)
	// Revoke 仅撤回发送者本人且仍正常的消息,返回受影响行数
	Revoke(ctx context.Context, conversationID, seq, senderID int64) (int64, error)
}
