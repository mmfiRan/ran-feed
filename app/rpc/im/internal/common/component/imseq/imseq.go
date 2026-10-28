// Package imseq 会话内序号分配与 max_seq 水位的唯一 owner
package imseq

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"ran-feed/app/rpc/im/internal/repositories"
)

// Allocator 序号分配器
type Allocator struct{}

// New 构造序号分配器
func New() *Allocator {
	return &Allocator{}
}

// Malloc 在会话行锁内分配下一个序号并推进水位
// repo 由调用方按事务构造,本组件不拥有事务边界
func (a *Allocator) Malloc(ctx context.Context, repo repositories.ConversationRepository, conversationID, lastMsgID, senderID int64, lastMsgAt time.Time) (int64, error) {
	if repo == nil || conversationID <= 0 || lastMsgID <= 0 {
		return 0, fmt.Errorf("imseq 参数非法 conversationID=%d lastMsgID=%d", conversationID, lastMsgID)
	}

	conv, err := repo.GetByIDForUpdate(ctx, conversationID)
	if err != nil {
		return 0, err
	}
	if conv == nil {
		return 0, gorm.ErrRecordNotFound
	}

	seq := conv.MaxSeq + 1
	if err := repo.UpdateSeqWatermark(ctx, conversationID, seq, lastMsgID, senderID, lastMsgAt); err != nil {
		return 0, err
	}
	return seq, nil
}
