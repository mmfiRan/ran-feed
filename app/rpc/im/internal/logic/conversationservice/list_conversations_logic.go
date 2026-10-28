package conversationservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/convert"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/app/rpc/im/internal/repositories"
	"ran-feed/app/rpc/im/internal/svc"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/utils"
)

// ListConversationsLogic 会话列表
type ListConversationsLogic struct {
	ctx context.Context
	logx.Logger
	conversationRepo repositories.ConversationRepository
	memberRepo       repositories.MemberRepository
	messageRepo      repositories.MessageRepository
	groupRepo        repositories.GroupRepository
}

// NewListConversationsLogic 构造
func NewListConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListConversationsLogic {
	return &ListConversationsLogic{
		ctx:              ctx,
		Logger:           logx.WithContext(ctx),
		conversationRepo: svcCtx.ConversationRepository,
		memberRepo:       svcCtx.MemberRepository,
		messageRepo:      svcCtx.MessageRepository,
		groupRepo:        svcCtx.GroupRepository,
	}
}

// ListConversations 按 (last_msg_at, id) 游标倒序返回会话
func (l *ListConversationsLogic) ListConversations(in *im.ListConversationsReq) (*im.ListConversationsRes, error) {
	if in.UserId <= 0 {
		return nil, errorx.NewMsg("参数错误")
	}
	limit := utils.ClampPageSize(in.PageSize)
	cursorLastMsgAt := time.Time{}
	if in.CursorLastMsgAt > 0 {
		cursorLastMsgAt = time.UnixMilli(in.CursorLastMsgAt)
	}

	conversationIDs, err := l.conversationRepo.ListIDsByUser(l.ctx, in.UserId, cursorLastMsgAt, in.CursorId, limit+1)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话列表失败"))
	}
	hasMore := len(conversationIDs) > limit
	if hasMore {
		conversationIDs = conversationIDs[:limit]
	}
	if len(conversationIDs) == 0 {
		return &im.ListConversationsRes{}, nil
	}

	conversationMap, err := l.conversationRepo.BatchGetByIDs(l.ctx, conversationIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话列表失败"))
	}
	memberMap, err := l.memberRepo.BatchGetByUser(l.ctx, in.UserId, conversationIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话成员失败"))
	}

	lastMsgIDs := make([]int64, 0, len(conversationIDs))
	groupIDs := make([]int64, 0, len(conversationIDs))
	for _, id := range conversationIDs {
		conv := conversationMap[id]
		if conv == nil {
			continue
		}
		if conv.LastMsgID > 0 {
			lastMsgIDs = append(lastMsgIDs, conv.LastMsgID)
		}
		if imEnum.ConvTypeEnum(conv.ConvType) == imEnum.ConvTypeGroup && conv.OwnerID > 0 {
			groupIDs = append(groupIDs, conv.OwnerID)
		}
	}
	messageMap, err := l.messageRepo.BatchGetByIDs(l.ctx, lastMsgIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询会话消息失败"))
	}
	groupMap, err := l.groupRepo.BatchGetByIDs(l.ctx, groupIDs)
	if err != nil {
		return nil, errorx.Wrap(l.ctx, err, errorx.NewMsg("查询群信息失败"))
	}

	items := make([]*im.ConversationItem, 0, len(conversationIDs))
	for _, id := range conversationIDs {
		conv := conversationMap[id]
		member := memberMap[id]
		if conv == nil || member == nil {
			continue
		}
		var item *im.ConversationItem
		switch imEnum.ConvTypeEnum(conv.ConvType) {
		case imEnum.ConvTypeSingle:
			item = convert.SingleConversationItemFromModel(conv, member, messageMap[conv.LastMsgID], in.UserId)
		case imEnum.ConvTypeGroup:
			item = convert.GroupConversationItemFromModel(conv, member, messageMap[conv.LastMsgID], conv.OwnerID, groupMap[conv.OwnerID])
		}
		if item != nil {
			items = append(items, item)
		}
	}

	resp := &im.ListConversationsRes{Items: items}
	if len(items) > 0 {
		lastConv := conversationMap[items[len(items)-1].ConversationId]
		resp.NextCursorLastMsgAt = lastMsgAtMillis(lastConv)
		resp.NextCursorId = lastConv.ID
		resp.HasMore = hasMore
	}
	return resp, nil
}

// lastMsgAtMillis 空会话以创建时间兜底,保证游标可用
func lastMsgAtMillis(conv *model.RanFeedImConversation) int64 {
	if conv == nil {
		return 0
	}
	if conv.LastMsgAt != nil {
		return conv.LastMsgAt.UnixMilli()
	}
	return conv.CreatedAt.UnixMilli()
}
