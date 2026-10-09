package contentservicelogic

import (
	"testing"

	contentEnum "ran-feed/app/rpc/content/internal/common/enums"

	"github.com/stretchr/testify/assert"
)

// TestBuildContentModel 统一构造主行 审计字段与归属一致
func TestBuildContentModel(t *testing.T) {
	row := buildContentModel(7, contentEnum.ContentTypeArticle, contentEnum.ContentStatusDraft, contentEnum.VisibilityPublic)
	assert.NotZero(t, row.ID)
	assert.Equal(t, int64(7), row.UserID)
	assert.Equal(t, contentEnum.ContentTypeArticle.Int32(), row.ContentType)
	assert.Equal(t, contentEnum.ContentStatusDraft.Int32(), row.Status)
	assert.Equal(t, contentEnum.VisibilityPublic.Int32(), row.Visibility)
	assert.Equal(t, int64(7), row.CreatedBy)
	assert.Equal(t, int64(7), row.UpdatedBy)
}
