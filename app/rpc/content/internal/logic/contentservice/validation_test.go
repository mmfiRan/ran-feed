package contentservicelogic

import (
	"testing"

	"ran-feed/app/rpc/content/content"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveWriteVisibility 发布必须显式合法可见性 草稿缺省回退 非法值一律拦下
func TestResolveWriteVisibility(t *testing.T) {
	_, err := resolveWriteVisibility(writeModePublish, content.Visibility_VISIBILITY_UNSPECIFIED, 0)
	require.Error(t, err, "发布缺省可见性应报错 不得写入 0")

	v, err := resolveWriteVisibility(writeModePublish, content.Visibility_VISIBILITY_PRIVATE, 0)
	require.NoError(t, err)
	assert.Equal(t, int32(content.Visibility_VISIBILITY_PRIVATE), v)

	v, err = resolveWriteVisibility(writeModeDraft, content.Visibility_VISIBILITY_UNSPECIFIED, int32(content.Visibility_VISIBILITY_PRIVATE))
	require.NoError(t, err)
	assert.Equal(t, int32(content.Visibility_VISIBILITY_PRIVATE), v, "草稿缺省回退 fallback")

	_, err = resolveWriteVisibility(writeModeDraft, content.Visibility(99), 10)
	require.Error(t, err, "非法可见性应拦下")
}

// TestPublishValidators 发布完整性校验 RPC 层兜底
func TestPublishValidators(t *testing.T) {
	assert.Error(t, validateArticlePublish("", "cover", "body"))
	assert.Error(t, validateArticlePublish("title", "", "body"))
	assert.Error(t, validateArticlePublish("title", "cover", ""))
	assert.NoError(t, validateArticlePublish("title", "cover", "body"))

	assert.Error(t, validateVideoPublish("", "cover", "video"))
	assert.Error(t, validateVideoPublish("title", "", "video"))
	assert.Error(t, validateVideoPublish("title", "cover", ""))
	assert.NoError(t, validateVideoPublish("title", "cover", "video"))
}
