package enums

import (
	"testing"

	"ran-feed/app/rpc/notification/notification"

	"github.com/stretchr/testify/assert"
)

// TestEnumValuesMatchProto 业务枚举取值必须与 proto 枚举一致
// 落库用业务枚举 列表返回与类型过滤用 proto 漂移会让过滤不到自己的通知
func TestEnumValuesMatchProto(t *testing.T) {
	tests := []struct {
		name  string
		biz   int32
		proto int32
	}{
		{"通知类型 未指定", NotifyTypeUnknown.Int32(), int32(notification.NotifyType_NOTIFY_TYPE_UNSPECIFIED)},
		{"通知类型 赞或收藏", NotifyTypeLikeFavorite.Int32(), int32(notification.NotifyType_NOTIFY_TYPE_LIKE_FAVORITE)},
		{"通知类型 评论或回复", NotifyTypeCommentReply.Int32(), int32(notification.NotifyType_NOTIFY_TYPE_COMMENT_REPLY)},
		{"通知类型 关注", NotifyTypeFollow.Int32(), int32(notification.NotifyType_NOTIFY_TYPE_FOLLOW)},
		{"通知类型 内容审核结果", NotifyTypeContentReview.Int32(), int32(notification.NotifyType_NOTIFY_TYPE_CONTENT_REVIEW)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.proto, tt.biz)
		})
	}
}
