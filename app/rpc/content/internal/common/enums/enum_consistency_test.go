package enums

import (
	"testing"

	"ran-feed/app/rpc/content/content"

	"github.com/stretchr/testify/assert"
)

// TestEnumValuesMatchProto 业务枚举取值必须与 proto 枚举一致
// DB 用业务枚举 RPC 契约用 proto 两边漂移会写出对方不认的值 这里先失败
func TestEnumValuesMatchProto(t *testing.T) {
	tests := []struct {
		name  string
		biz   int32
		proto int32
	}{
		{"内容状态 未指定", ContentStatusUnknown.Int32(), int32(content.ContentStatus_CONTENT_STATUS_UNSPECIFIED)},
		{"内容状态 草稿", ContentStatusDraft.Int32(), int32(content.ContentStatus_CONTENT_STATUS_DRAFT)},
		{"内容状态 处理中", ContentStatusProcessing.Int32(), int32(content.ContentStatus_CONTENT_STATUS_PROCESSING)},
		{"内容状态 已发布", ContentStatusPublished.Int32(), int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED)},
		{"内容状态 失败", ContentStatusFailed.Int32(), int32(content.ContentStatus_CONTENT_STATUS_FAILED)},
		{"内容状态 下架", ContentStatusTakenDown.Int32(), int32(content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN)},
		{"内容状态 待审", ContentStatusPendingReview.Int32(), int32(content.ContentStatus_CONTENT_STATUS_PENDING_REVIEW)},
		{"内容状态 拒绝", ContentStatusRejected.Int32(), int32(content.ContentStatus_CONTENT_STATUS_REJECTED)},

		{"内容类型 未指定", ContentTypeUnknown.Int32(), int32(content.ContentType_CONTENT_TYPE_UNSPECIFIED)},
		{"内容类型 文章", ContentTypeArticle.Int32(), int32(content.ContentType_CONTENT_TYPE_ARTICLE)},
		{"内容类型 视频", ContentTypeVideo.Int32(), int32(content.ContentType_CONTENT_TYPE_VIDEO)},

		{"可见性 未指定", VisibilityUnknown.Int32(), int32(content.Visibility_VISIBILITY_UNSPECIFIED)},
		{"可见性 公开", VisibilityPublic.Int32(), int32(content.Visibility_VISIBILITY_PUBLIC)},
		{"可见性 私密", VisibilityPrivate.Int32(), int32(content.Visibility_VISIBILITY_PRIVATE)},

		{"审核决策 未指定", ReviewDecisionUnknown.Int32(), int32(content.ReviewDecision_REVIEW_DECISION_UNSPECIFIED)},
		{"审核决策 通过", ReviewDecisionApprove.Int32(), int32(content.ReviewDecision_REVIEW_DECISION_APPROVE)},
		{"审核决策 拒绝", ReviewDecisionReject.Int32(), int32(content.ReviewDecision_REVIEW_DECISION_REJECT)},
		{"审核决策 下架", ReviewDecisionTakenDown.Int32(), int32(content.ReviewDecision_REVIEW_DECISION_TAKEN_DOWN)},
		{"审核决策 恢复", ReviewDecisionRestored.Int32(), int32(content.ReviewDecision_REVIEW_DECISION_RESTORED)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.proto, tt.biz)
		})
	}
}
