package admincontentservicelogic

import (
	"testing"

	"ran-feed/app/rpc/content/content"

	"github.com/stretchr/testify/assert"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
)

// TestReviewDecisionFor 下架与恢复都落审核记录且决策取值正确 便于管理端查历史
func TestReviewDecisionFor(t *testing.T) {
	assert.Equal(t,
		contentEnum.ReviewDecisionTakenDown,
		reviewDecisionFor(content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN))
	assert.Equal(t,
		contentEnum.ReviewDecisionRestored,
		reviewDecisionFor(content.ContentStatus_CONTENT_STATUS_PUBLISHED))
}
