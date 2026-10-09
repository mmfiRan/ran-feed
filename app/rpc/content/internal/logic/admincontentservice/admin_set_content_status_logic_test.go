package admincontentservicelogic

import (
	"testing"

	contentEnum "ran-feed/app/rpc/content/internal/common/enums"

	"github.com/stretchr/testify/assert"
)

// TestReviewDecisionFor 下架与恢复都落审核记录且决策取值正确 便于管理端查历史
func TestReviewDecisionFor(t *testing.T) {
	assert.Equal(t,
		contentEnum.ReviewDecisionTakenDown,
		reviewDecisionFor(contentEnum.ContentStatusTakenDown))
	assert.Equal(t,
		contentEnum.ReviewDecisionRestored,
		reviewDecisionFor(contentEnum.ContentStatusPublished))
}
