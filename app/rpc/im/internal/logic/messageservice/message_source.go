package messageservicelogic

import (
	"context"

	"ran-feed/app/rpc/im/internal/common/consts"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/pkg/errorx"
)

// ensureGroupWritable 群会话已解散时禁止收发
func ensureGroupWritable(ctx context.Context, groupRepo repositories.GroupRepository, conv *model.RanFeedImConversation) error {
	if conv == nil || imEnum.ConvTypeEnum(conv.ConvType) != imEnum.ConvTypeGroup || conv.OwnerID <= 0 {
		return nil
	}
	group, err := groupRepo.GetByID(ctx, conv.OwnerID)
	if err != nil {
		return err
	}
	if group == nil {
		return errorx.NewMsg("群不存在")
	}
	if imEnum.GroupStatusEnum(group.Status) == imEnum.GroupStatusDissolved {
		return consts.ErrGroupDissolved
	}
	return nil
}

// activePushTargets 活跃且未免打扰的成员
func activePushTargets(ctx context.Context, repo repositories.MemberRepository, conversationID int64) ([]int64, error) {
	members, err := repo.ListByConversation(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	targets := make([]int64, 0, len(members))
	for _, m := range members {
		if m.Muted == imEnum.MutedYes.Int32() {
			continue
		}
		targets = append(targets, m.UserID)
	}
	return targets, nil
}
