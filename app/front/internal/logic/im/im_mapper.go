package im

import (
	"strconv"
	"strings"

	frontutils "ran-feed/app/front/internal/common/utils"
	"ran-feed/app/front/internal/types"
	impb "ran-feed/app/rpc/im/im"
	userpb "ran-feed/app/rpc/user/user"
	"ran-feed/pkg/errorx"
)

// toIMMessagePreview pb 消息转 front 消息预览
func toIMMessagePreview(msg *impb.ImMessage) types.IMMessagePreview {
	if msg == nil {
		return types.IMMessagePreview{}
	}
	out := types.IMMessagePreview{
		MessageId:   msg.MessageId,
		Seq:         msg.Seq,
		SenderId:    msg.SenderId,
		ContentType: frontutils.ToEnumValue(msg.ContentType),
		Content:     msg.Content,
		Status:      frontutils.ToEnumValue(msg.Status),
	}
	if ts := msg.GetSendTime(); ts != nil {
		out.SendTime = ts.AsTime().UnixMilli()
	}
	return out
}

// toIMConversationItem pb 会话项转 front 项并富化 peer
func toIMConversationItem(it *impb.ConversationItem, userMap map[int64]*userpb.UserInfo) types.IMConversationItem {
	out := types.IMConversationItem{
		ConversationId: it.ConversationId,
		ConvType:       frontutils.ToEnumValue(it.ConvType),
		MaxSeq:         it.MaxSeq,
		ReadSeq:        it.ReadSeq,
		MinSeq:         it.MinSeq,
		Unread:         it.Unread,
		Muted:          it.Muted,
	}
	if it.PeerUserId > 0 {
		peer := &types.IMUserBrief{UserId: it.PeerUserId}
		if u := userMap[it.PeerUserId]; u != nil {
			peer.Nickname = u.Nickname
			peer.Avatar = u.Avatar
		}
		out.Peer = peer
	}
	if it.GroupId > 0 {
		out.Group = &types.IMGroupBrief{
			GroupId: it.GroupId,
			Name:    it.GroupName,
			Avatar:  it.GroupAvatar,
			Status:  frontutils.ToEnumValue(it.GroupStatus),
		}
	}
	if it.LastMessage != nil {
		last := toIMMessagePreview(it.LastMessage)
		out.LastMessage = &last
	}
	return out
}

// collectPeerIDs 去重收集单聊对方用户 ID
func collectPeerIDs(items []*impb.ConversationItem) []int64 {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(items))
	out := make([]int64, 0, len(items))
	for _, it := range items {
		if it == nil || it.PeerUserId <= 0 {
			continue
		}
		if _, ok := seen[it.PeerUserId]; ok {
			continue
		}
		seen[it.PeerUserId] = struct{}{}
		out = append(out, it.PeerUserId)
	}
	return out
}

// parseIMUserIDs 字符串用户 ID 列表转 int64,非法即报错不静默丢弃
func parseIMUserIDs(raw []string) ([]int64, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	seen := make(map[int64]struct{}, len(raw))
	out := make([]int64, 0, len(raw))
	for _, s := range raw {
		userID, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil || userID <= 0 {
			return nil, errorx.NewMsg("成员ID非法")
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		out = append(out, userID)
	}
	return out, nil
}
