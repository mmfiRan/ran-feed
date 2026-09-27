package contentservicelogic

import (
	"testing"

	"ran-feed/app/rpc/content/content"

	"github.com/stretchr/testify/assert"
)

// TestBuildContentModel 统一构造主行 审计字段与归属一致
func TestBuildContentModel(t *testing.T) {
	row := buildContentModel(7, content.ContentType_CONTENT_TYPE_ARTICLE, content.ContentStatus_CONTENT_STATUS_DRAFT, int32(content.Visibility_VISIBILITY_PUBLIC))
	assert.NotZero(t, row.ID)
	assert.Equal(t, int64(7), row.UserID)
	assert.Equal(t, int32(content.ContentType_CONTENT_TYPE_ARTICLE), row.ContentType)
	assert.Equal(t, int32(content.ContentStatus_CONTENT_STATUS_DRAFT), row.Status)
	assert.Equal(t, int32(content.Visibility_VISIBILITY_PUBLIC), row.Visibility)
	assert.Equal(t, int64(7), row.CreatedBy)
	assert.Equal(t, int64(7), row.UpdatedBy)
}
