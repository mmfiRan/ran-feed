// Package convert IM 域模型与 protobuf 的边界转换
package convert

import (
	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/consts"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/entity/model"
	"ran-feed/pkg/enums"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// MessageFromModel 消息模型转 pb
func MessageFromModel(row *model.RanFeedImMessage) *im.ImMessage {
	if row == nil {
		return nil
	}
	return &im.ImMessage{
		MessageId:   row.ID,
		Seq:         row.Seq,
		SenderId:    row.SenderID,
		ContentType: enums.ToCommonPB(imEnum.ContentTypeEnum(row.ContentType)),
		Content:     row.Content,
		Status:      enums.ToCommonPB(imEnum.MessageStatusEnum(row.Status)),
		SendTime:    timestamppb.New(row.CreatedAt),
	}
}

// MessagesFromModels 消息模型批量转 pb,保持入参顺序
func MessagesFromModels(rows []*model.RanFeedImMessage) []*im.ImMessage {
	if len(rows) == 0 {
		return nil
	}
	out := make([]*im.ImMessage, 0, len(rows))
	for _, row := range rows {
		if msg := MessageFromModel(row); msg != nil {
			out = append(out, msg)
		}
	}
	return out
}

// SingleConversationItemFromModel 单聊会话项,对方用户 ID 由 biz_key 解出
func SingleConversationItemFromModel(conv *model.RanFeedImConversation, member *model.RanFeedImConversationMember, last *model.RanFeedImMessage, userID int64) *im.ConversationItem {
	item := baseConversationItem(conv, member, last)
	if item == nil {
		return nil
	}
	item.PeerUserId = peerUserID(conv.BizKey, userID)
	return item
}

// GroupConversationItemFromModel 群聊会话项,群资料缺失时只回群 ID
func GroupConversationItemFromModel(conv *model.RanFeedImConversation, member *model.RanFeedImConversationMember, last *model.RanFeedImMessage, groupID int64, group *model.RanFeedImGroup) *im.ConversationItem {
	item := baseConversationItem(conv, member, last)
	if item == nil {
		return nil
	}
	item.GroupId = groupID
	if group != nil {
		item.GroupName = group.Name
		item.GroupAvatar = group.Avatar
		item.GroupStatus = enums.ToCommonPB(imEnum.GroupStatusEnum(group.Status))
	}
	return item
}

// baseConversationItem 会话列表项共有的字段
func baseConversationItem(conv *model.RanFeedImConversation, member *model.RanFeedImConversationMember, last *model.RanFeedImMessage) *im.ConversationItem {
	if conv == nil || member == nil {
		return nil
	}
	return &im.ConversationItem{
		ConversationId: conv.ID,
		ConvType:       enums.ToCommonPB(imEnum.ConvTypeEnum(conv.ConvType)),
		MaxSeq:         conv.MaxSeq,
		ReadSeq:        member.ReadSeq,
		MinSeq:         member.MinSeq,
		Unread:         unreadCount(conv.MaxSeq, member.ReadSeq),
		Muted:          imEnum.MutedEnum(member.Muted) == imEnum.MutedYes,
		LastMessage:    MessageFromModel(last),
	}
}

// unreadCount 未读数由两个水位相减
func unreadCount(maxSeq, readSeq int64) int64 {
	if maxSeq <= readSeq {
		return 0
	}
	return maxSeq - readSeq
}

// peerUserID 从单聊幂等键解出对方用户 ID
func peerUserID(bizKey string, userID int64) int64 {
	a, b, ok := consts.ParseSingleConversationBizKey(bizKey)
	if !ok {
		return 0
	}
	if a == userID {
		return b
	}
	return a
}
