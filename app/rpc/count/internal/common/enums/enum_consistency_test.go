package enums

import (
	"testing"

	"ran-feed/app/rpc/count/count"

	"github.com/stretchr/testify/assert"
)

// TestEnumValuesMatchProto 业务枚举取值必须与 proto 枚举一致
// canal 行解析与 RPC 请求共用同一套值 漂移会让计数落到错误的桶
func TestEnumValuesMatchProto(t *testing.T) {
	tests := []struct {
		name  string
		biz   int32
		proto int32
	}{
		{"计数业务 未指定", BizTypeUnknown.Int32(), int32(count.BizType_BIZ_TYPE_UNSPECIFIED)},
		{"计数业务 点赞", BizTypeLike.Int32(), int32(count.BizType_BIZ_TYPE_LIKE)},
		{"计数业务 收藏", BizTypeFavorite.Int32(), int32(count.BizType_BIZ_TYPE_FAVORITE)},
		{"计数业务 评论", BizTypeComment.Int32(), int32(count.BizType_BIZ_TYPE_COMMENT)},
		{"计数业务 粉丝", BizTypeFollowed.Int32(), int32(count.BizType_BIZ_TYPE_FOLLOWED)},
		{"计数业务 关注", BizTypeFollowing.Int32(), int32(count.BizType_BIZ_TYPE_FOLLOWING)},

		{"计数对象 未指定", TargetTypeUnknown.Int32(), int32(count.TargetType_TARGET_TYPE_UNSPECIFIED)},
		{"计数对象 内容", TargetTypeContent.Int32(), int32(count.TargetType_TARGET_TYPE_CONTENT)},
		{"计数对象 用户", TargetTypeUser.Int32(), int32(count.TargetType_TARGET_TYPE_USER)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.proto, tt.biz)
		})
	}
}
