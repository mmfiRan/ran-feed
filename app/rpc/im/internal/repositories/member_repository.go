package repositories

import (
	"context"

	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
)

// MemberRepository 会话成员与阅读游标数据层
type MemberRepository interface {
	WithTx(tx *query.Query) MemberRepository
	// Upsert 幂等写成员,撞 uk_conv_user 时恢复 is_deleted 并更新修改人
	Upsert(ctx context.Context, rows []*model.RanFeedImConversationMember) error
	// UpsertJoin 入群写成员,撞 uk_conv_user 时同步 min_seq 与 read_seq
	UpsertJoin(ctx context.Context, rows []*model.RanFeedImConversationMember) error
	// Get 查不到返回 (nil, nil)
	Get(ctx context.Context, conversationID, userID int64) (*model.RanFeedImConversationMember, error)
	// BatchGetByUser 取某用户在一批会话下的成员记录
	BatchGetByUser(ctx context.Context, userID int64, conversationIDs []int64) (map[int64]*model.RanFeedImConversationMember, error)
	// ListByConversation 取会话下全部活跃成员
	ListByConversation(ctx context.Context, conversationID int64) ([]*model.RanFeedImConversationMember, error)
	// AdvanceReadSeq 仅当 read_seq 更小时推进,返回受影响行数
	AdvanceReadSeq(ctx context.Context, conversationID, userID, readSeq int64) (int64, error)
	// MarkLeft 标记成员退群,返回受影响行数
	MarkLeft(ctx context.Context, conversationID, userID int64) (int64, error)
}
