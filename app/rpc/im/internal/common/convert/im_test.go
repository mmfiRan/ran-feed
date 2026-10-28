package convert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ran-feed/app/rpc/im/im"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/app/rpc/im/internal/entity/model"
)

func TestUnreadCount(t *testing.T) {
	assert.Equal(t, int64(2), unreadCount(5, 3))
	assert.Equal(t, int64(0), unreadCount(3, 5))
	assert.Equal(t, int64(0), unreadCount(3, 3))
}

func TestPeerUserID(t *testing.T) {
	assert.Equal(t, int64(2), peerUserID("S:1_2", 1))
	assert.Equal(t, int64(1), peerUserID("S:1_2", 2))
	assert.Equal(t, int64(0), peerUserID("bad", 1))
}

func TestConversationItemFromModel(t *testing.T) {
	member := &model.RanFeedImConversationMember{ReadSeq: 3, MinSeq: 0, Muted: imEnum.MutedNo.Int32()}
	last := &model.RanFeedImMessage{
		ID:          9,
		Seq:         5,
		SenderID:    2,
		ContentType: imEnum.ContentTypeText.Int32(),
		Content:     "hi",
		Status:      imEnum.MessageStatusNormal.Int32(),
	}

	singleConv := &model.RanFeedImConversation{ID: 10, ConvType: imEnum.ConvTypeSingle.Int32(), BizKey: "S:1_2", MaxSeq: 5}
	single := SingleConversationItemFromModel(singleConv, member, last, 1)
	require.NotNil(t, single)
	assert.Equal(t, int64(10), single.ConversationId)
	assert.Equal(t, int32(im.ConvType_CONV_TYPE_SINGLE), single.ConvType.GetCode())
	assert.Equal(t, int64(2), single.PeerUserId)
	assert.Equal(t, int64(2), single.Unread)
	require.NotNil(t, single.LastMessage)
	assert.Equal(t, int32(im.MessageContentType_MESSAGE_CONTENT_TYPE_TEXT), single.LastMessage.GetContentType().GetCode())
	assert.Equal(t, int32(im.MessageStatus_MESSAGE_STATUS_NORMAL), single.LastMessage.GetStatus().GetCode())

	groupConv := &model.RanFeedImConversation{ID: 11, ConvType: imEnum.ConvTypeGroup.Int32(), BizKey: "G:77", MaxSeq: 6, OwnerID: 77}
	group := &model.RanFeedImGroup{ID: 77, Name: "g", Avatar: "a.png", Status: imEnum.GroupStatusNormal.Int32()}
	grouped := GroupConversationItemFromModel(groupConv, member, nil, 77, group)
	require.NotNil(t, grouped)
	assert.Equal(t, int64(77), grouped.GroupId)
	assert.Equal(t, "g", grouped.GroupName)
	assert.Equal(t, int32(im.GroupStatus_GROUP_STATUS_NORMAL), grouped.GroupStatus.GetCode())
	assert.Nil(t, grouped.LastMessage)

	// 群资料缺失只回群 ID 不拖垮整行
	missing := GroupConversationItemFromModel(groupConv, member, nil, 77, nil)
	require.NotNil(t, missing)
	assert.Equal(t, int64(77), missing.GroupId)
	assert.Empty(t, missing.GroupName)
	assert.Equal(t, int32(im.GroupStatus_GROUP_STATUS_UNSPECIFIED), missing.GroupStatus.GetCode())

	// 会话或成员缺失整行跳过
	assert.Nil(t, SingleConversationItemFromModel(nil, member, nil, 1))
	assert.Nil(t, SingleConversationItemFromModel(singleConv, nil, nil, 1))
	assert.Nil(t, GroupConversationItemFromModel(nil, member, nil, 77, group))
}
