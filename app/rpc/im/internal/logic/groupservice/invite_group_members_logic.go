package groupservicelogic

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/protobuf/types/known/emptypb"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/consts"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/entity/query"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/app/rpc/im/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"
)

// InviteGroupMembersLogic 邀请成员入群
type InviteGroupMembersLogic struct {
	ctx context.Context
	logx.Logger
	conversationRepo repositories.ConversationRepository
	memberRepo       repositories.MemberRepository
	groupRepo        repositories.GroupRepository
	maxMemberCount   int
}

// NewInviteGroupMembersLogic 构造
func NewInviteGroupMembersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InviteGroupMembersLogic {
	return &InviteGroupMembersLogic{
		ctx:              ctx,
		Logger:           logx.WithContext(ctx),
		conversationRepo: svcCtx.ConversationRepository,
		memberRepo:       svcCtx.MemberRepository,
		groupRepo:        svcCtx.GroupRepository,
		maxMemberCount:   svcCtx.Config.Group.MaxMemberCount,
	}
}

// InviteGroupMembers 仅群主可邀请,新成员可见起点为当前 max_seq
func (l *InviteGroupMembersLogic) InviteGroupMembers(in *im.InviteGroupMembersReq) (*emptypb.Empty, error) {
	if in.UserId <= 0 || in.GroupId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	memberIDs, err := sanitizeMemberIDs(in.UserId, in.MemberIds)
	if err != nil {
		return nil, err
	}
	err = query.Q.Transaction(func(tx *query.Query) error {
		groupRepo := l.groupRepo.WithTx(tx)
		group, err := groupRepo.GetByIDForUpdate(l.ctx, in.GroupId)
		if err != nil {
			return err
		}
		if group == nil {
			return errorx.NewMsg("群不存在")
		}
		if imEnum.GroupStatusEnum(group.Status) == imEnum.GroupStatusDissolved {
			return consts.ErrGroupDissolved
		}
		if group.OwnerID != in.UserId {
			return errorx.NewMsg("只有群主可以邀请成员")
		}

		conv, err := l.conversationRepo.WithTx(tx).GetByBizKey(l.ctx, consts.GroupConversationBizKey(in.GroupId))
		if err != nil {
			return err
		}
		if conv == nil {
			return errorx.NewMsg("群会话不存在")
		}

		memberRepo := l.memberRepo.WithTx(tx)
		active, err := memberRepo.ListByConversation(l.ctx, conv.ID)
		if err != nil {
			return err
		}
		activeSet := make(map[int64]struct{}, len(active))
		for _, member := range active {
			activeSet[member.UserID] = struct{}{}
		}

		remaining := make([]int64, 0, len(memberIDs))
		for _, userID := range memberIDs {
			if _, ok := activeSet[userID]; ok {
				continue
			}
			remaining = append(remaining, userID)
		}
		if len(remaining) == 0 {
			return nil
		}
		if len(active)+len(remaining) > l.maxMemberCount {
			return consts.ErrGroupFull
		}

		now := time.Now()
		rows := make([]*model.RanFeedImConversationMember, 0, len(remaining))
		for _, userID := range remaining {
			rows = append(rows, &model.RanFeedImConversationMember{
				ID:             snowflake.GenID(),
				ConversationID: conv.ID,
				UserID:         userID,
				ReadSeq:        conv.MaxSeq,
				MinSeq:         conv.MaxSeq,
				Muted:          imEnum.MutedNo.Int32(),
				CreatedBy:      in.UserId,
				UpdatedBy:      in.UserId,
				CreatedAt:      now,
				UpdatedAt:      now,
			})
		}
		if err := memberRepo.UpsertJoin(l.ctx, rows); err != nil {
			return err
		}
		return groupRepo.UpdateMemberCount(l.ctx, in.GroupId, int32(len(active)+len(remaining)), in.UserId)
	})
	if err != nil {
		var bizErr *errorx.BizError
		if errors.As(err, &bizErr) {
			return nil, bizErr
		}
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("邀请成员失败"))
	}
	return &emptypb.Empty{}, nil
}
