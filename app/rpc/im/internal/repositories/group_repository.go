package repositories

import (
	"context"

	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
)

// GroupRepository 群元信息数据层
type GroupRepository interface {
	WithTx(tx *query.Query) GroupRepository
	// Create 新建群
	Create(ctx context.Context, row *model.RanFeedImGroup) error
	// GetByID 查不到返回 (nil, nil)
	GetByID(ctx context.Context, groupID int64) (*model.RanFeedImGroup, error)
	// GetByIDForUpdate 行锁查群,用于邀请与退群的成员数串行化
	GetByIDForUpdate(ctx context.Context, groupID int64) (*model.RanFeedImGroup, error)
	// BatchGetByIDs 批量取群
	BatchGetByIDs(ctx context.Context, groupIDs []int64) (map[int64]*model.RanFeedImGroup, error)
	// UpdateMemberCount 更新成员数
	UpdateMemberCount(ctx context.Context, groupID int64, memberCount int32, updatedBy int64) error
	// Dissolve 正常群置为已解散,幂等
	Dissolve(ctx context.Context, groupID, updatedBy int64) error
}
