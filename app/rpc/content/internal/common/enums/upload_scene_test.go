package enums

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUploadSceneIMPath(t *testing.T) {
	assert.True(t, UploadSceneImImage.Valid())
	assert.True(t, UploadSceneImVideo.Valid())
	assert.Equal(t, "im/image", UploadSceneImImage.Path())
	assert.Equal(t, "im/video", UploadSceneImVideo.Path())
	assert.Equal(t, int32(4), UploadSceneImImage.Int32())
	assert.Equal(t, int32(5), UploadSceneImVideo.Int32())
}
