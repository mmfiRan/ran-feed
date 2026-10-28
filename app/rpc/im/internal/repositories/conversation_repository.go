package repositories

import (
	"context"
	"time"

	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
)

// ConversationRepository 会话主表数据层
type ConversationRepository interface {
	WithTx(tx *query.Query) ConversationRepository
	// Create 新建会话,biz_key 冲突返回 gorm.ErrDuplicatedKey
	Create(ctx context.Context, row *model.RanFeedImConversation) error
	// GetByID 查不到返回 (nil, nil)
	GetByID(ctx context.Context, conversationID int64) (*model.RanFeedImConversation, error)
	// GetByIDForUpdate 行锁查会话,用于 seq 分配串行化
	GetByIDForUpdate(ctx context.Context, conversationID int64) (*model.RanFeedImConversation, error)
	// GetByBizKey 查不到返回 (nil, nil)
	GetByBizKey(ctx context.Context, bizKey string) (*model.RanFeedImConversation, error)
	// UpdateSeqWatermark 推进 max_seq 并刷新最后消息字段
	UpdateSeqWatermark(ctx context.Context, conversationID, seq, lastMsgID, senderID int64, lastMsgAt time.Time) error
	// BatchGetByIDs 按会话 ID 批量取
	BatchGetByIDs(ctx context.Context, conversationIDs []int64) (map[int64]*model.RanFeedImConversation, error)
	// ListIDsByUser 用户会话 ID 列表,按 (last_msg_at, id) 倒序,多取 1 条供调用方判 has_more
	ListIDsByUser(ctx context.Context, userID int64, cursorLastMsgAt time.Time, cursorID int64, limit int) ([]int64, error)
}
