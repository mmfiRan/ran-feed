package enums

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/im/im"
)

func TestConvTypeEnumMatchesProto(t *testing.T) {
	assert.Equal(t, int32(im.ConvType_CONV_TYPE_UNSPECIFIED), ConvTypeUnknown.Int32())
	assert.Equal(t, int32(im.ConvType_CONV_TYPE_SINGLE), ConvTypeSingle.Int32())
	assert.Equal(t, int32(im.ConvType_CONV_TYPE_GROUP), ConvTypeGroup.Int32())
}

func TestContentTypeEnumMatchesProto(t *testing.T) {
	assert.Equal(t, int32(im.MessageContentType_MESSAGE_CONTENT_TYPE_UNSPECIFIED), ContentTypeUnknown.Int32())
	assert.Equal(t, int32(im.MessageContentType_MESSAGE_CONTENT_TYPE_TEXT), ContentTypeText.Int32())
	assert.Equal(t, int32(im.MessageContentType_MESSAGE_CONTENT_TYPE_IMAGE), ContentTypeImage.Int32())
	assert.Equal(t, int32(im.MessageContentType_MESSAGE_CONTENT_TYPE_VIDEO), ContentTypeVideo.Int32())
}

func TestMessageStatusEnumMatchesProto(t *testing.T) {
	assert.Equal(t, int32(im.MessageStatus_MESSAGE_STATUS_UNSPECIFIED), MessageStatusUnknown.Int32())
	assert.Equal(t, int32(im.MessageStatus_MESSAGE_STATUS_NORMAL), MessageStatusNormal.Int32())
	assert.Equal(t, int32(im.MessageStatus_MESSAGE_STATUS_REVOKED), MessageStatusRevoked.Int32())
}

func TestGroupStatusEnumMatchesProto(t *testing.T) {
	assert.Equal(t, int32(im.GroupStatus_GROUP_STATUS_UNSPECIFIED), GroupStatusUnknown.Int32())
	assert.Equal(t, int32(im.GroupStatus_GROUP_STATUS_NORMAL), GroupStatusNormal.Int32())
	assert.Equal(t, int32(im.GroupStatus_GROUP_STATUS_DISSOLVED), GroupStatusDissolved.Int32())
}

func TestMutedEnum(t *testing.T) {
	assert.Equal(t, int32(0), MutedNo.Int32())
	assert.Equal(t, int32(1), MutedYes.Int32())
	assert.True(t, MutedNo.Valid())
	assert.True(t, MutedYes.Valid())
}
