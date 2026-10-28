package im

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	impb "ran-feed/app/rpc/im/im"
	userpb "ran-feed/app/rpc/user/user"
	"ran-feed/pkg/commonpb"
)

func TestCollectPeerIDs(t *testing.T) {
	items := []*impb.ConversationItem{
		{PeerUserId: 2},
		{PeerUserId: 2},
		{PeerUserId: 3},
		{PeerUserId: 0},
		nil,
	}
	assert.Equal(t, []int64{2, 3}, collectPeerIDs(items))
	assert.Nil(t, collectPeerIDs(nil))
}

func TestToIMConversationItem(t *testing.T) {
	it := &impb.ConversationItem{
		ConversationId: 10,
		ConvType:       &commonpb.EnumValue{Code: int32(impb.ConvType_CONV_TYPE_SINGLE), Name: "SINGLE", Message: "单聊"},
		PeerUserId:     2,
		MaxSeq:         5,
		ReadSeq:        3,
		MinSeq:         0,
		Unread:         2,
		Muted:          false,
		LastMessage: &impb.ImMessage{
			MessageId:   9,
			Seq:         5,
			SenderId:    2,
			ContentType: &commonpb.EnumValue{Code: int32(impb.MessageContentType_MESSAGE_CONTENT_TYPE_TEXT), Name: "TEXT", Message: "文本"},
			Content:     "hi",
			Status:      &commonpb.EnumValue{Code: int32(impb.MessageStatus_MESSAGE_STATUS_NORMAL), Name: "NORMAL", Message: "正常"},
		},
	}
	userMap := map[int64]*userpb.UserInfo{2: {UserId: 2, Nickname: "peer", Avatar: "a.png"}}

	got := toIMConversationItem(it, userMap)
	assert.Equal(t, int64(10), got.ConversationId)
	assert.Equal(t, int32(10), got.ConvType.Code)
	assert.Equal(t, "单聊", got.ConvType.Message)
	require.NotNil(t, got.Peer)
	assert.Equal(t, int64(2), got.Peer.UserId)
	assert.Equal(t, "peer", got.Peer.Nickname)
	assert.Nil(t, got.Group)
	require.NotNil(t, got.LastMessage)
	assert.Equal(t, int64(5), got.LastMessage.Seq)
	assert.Equal(t, "hi", got.LastMessage.Content)
	assert.Equal(t, int32(10), got.LastMessage.ContentType.Code)

	// 群会话映射
	groupItem := &impb.ConversationItem{
		ConversationId: 11,
		ConvType:       &commonpb.EnumValue{Code: int32(impb.ConvType_CONV_TYPE_GROUP)},
		GroupId:        77,
		GroupName:      "g",
		GroupAvatar:    "g.png",
		GroupStatus:    &commonpb.EnumValue{Code: int32(impb.GroupStatus_GROUP_STATUS_NORMAL)},
	}
	groupGot := toIMConversationItem(groupItem, userMap)
	require.NotNil(t, groupGot.Group)
	assert.Equal(t, int64(77), groupGot.Group.GroupId)
	assert.Equal(t, "g", groupGot.Group.Name)
	assert.Equal(t, int32(10), groupGot.Group.Status.Code)
	assert.Nil(t, groupGot.Peer)
}

func TestParseIMUserIDs(t *testing.T) {
	got, err := parseIMUserIDs([]string{"2", " 3 ", "2"})
	require.NoError(t, err)
	assert.Equal(t, []int64{2, 3}, got)

	got, err = parseIMUserIDs(nil)
	require.NoError(t, err)
	assert.Nil(t, got)

	for _, bad := range [][]string{{"0"}, {"-1"}, {"abc"}, {""}} {
		_, err := parseIMUserIDs(bad)
		assert.Error(t, err, bad)
	}
}
