package groupservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

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

// CreateGroupLogic 创建群
type CreateGroupLogic struct {
	ctx context.Context
	logx.Logger
	conversationRepo repositories.ConversationRepository
	memberRepo       repositories.MemberRepository
	groupRepo        repositories.GroupRepository
	maxMemberCount   int
}

// NewCreateGroupLogic 构造
func NewCreateGroupLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateGroupLogic {
	return &CreateGroupLogic{
		ctx:              ctx,
		Logger:           logx.WithContext(ctx),
		conversationRepo: svcCtx.ConversationRepository,
		memberRepo:       svcCtx.MemberRepository,
		groupRepo:        svcCtx.GroupRepository,
		maxMemberCount:   svcCtx.Config.Group.MaxMemberCount,
	}
}

// CreateGroup 事务内建群,会话与初始成员一起落库
func (l *CreateGroupLogic) CreateGroup(in *im.CreateGroupReq) (*im.CreateGroupRes, error) {
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	name, err := validateGroupMeta(in.Name, in.Avatar)
	if err != nil {
		return nil, err
	}
	memberIDs, err := sanitizeMemberIDs(in.UserId, in.MemberIds)
	if err != nil {
		return nil, err
	}
	if 1+len(memberIDs) > l.maxMemberCount {
		return nil, consts.ErrGroupFull
	}

	groupID := snowflake.GenID()
	conversationID := snowflake.GenID()
	now := time.Now()
	err = query.Q.Transaction(func(tx *query.Query) error {
		group := &model.RanFeedImGroup{
			ID:          groupID,
			Name:        name,
			Avatar:      in.Avatar,
			OwnerID:     in.UserId,
			MemberCount: int32(1 + len(memberIDs)),
			Status:      imEnum.GroupStatusNormal.Int32(),
			CreatedBy:   in.UserId,
			UpdatedBy:   in.UserId,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := l.groupRepo.WithTx(tx).Create(l.ctx, group); err != nil {
			return err
		}

		conv := &model.RanFeedImConversation{
			ID:        conversationID,
			ConvType:  imEnum.ConvTypeGroup.Int32(),
			BizKey:    consts.GroupConversationBizKey(groupID),
			OwnerID:   groupID,
			MaxSeq:    0,
			LastMsgID: 0,
			LastMsgAt: &now,
			CreatedBy: in.UserId,
			UpdatedBy: in.UserId,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := l.conversationRepo.WithTx(tx).Create(l.ctx, conv); err != nil {
			return err
		}

		rows := make([]*model.RanFeedImConversationMember, 0, 1+len(memberIDs))
		for _, userID := range append([]int64{in.UserId}, memberIDs...) {
			rows = append(rows, &model.RanFeedImConversationMember{
				ID:             snowflake.GenID(),
				ConversationID: conversationID,
				UserID:         userID,
				ReadSeq:        0,
				MinSeq:         0,
				Muted:          imEnum.MutedNo.Int32(),
				CreatedBy:      in.UserId,
				UpdatedBy:      in.UserId,
				CreatedAt:      now,
				UpdatedAt:      now,
			})
		}
		return l.memberRepo.WithTx(tx).Upsert(l.ctx, rows)
	})
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("创建群失败"))
	}
	return &im.CreateGroupRes{GroupId: groupID, ConversationId: conversationID}, nil
}
