package contentevent

import (
	"testing"

	contentenums "ran-feed/pkg/enums/content"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContentEventConstructors 每种事件构造函数填对类型与字段
func TestContentEventConstructors(t *testing.T) {
	cases := []struct {
		name   string
		evt    *ContentEvent
		want   contentenums.EventTypeEnum
		reason string
	}{
		{"发布", NewContentPublishedEvent(1, 2), contentenums.EventTypePublished, ""},
		{"拒绝", NewContentRejectedEvent(1, 2, "违规"), contentenums.EventTypeRejected, "违规"},
		{"删除", NewContentDeletedEvent(1, 2), contentenums.EventTypeDeleted, ""},
		{"下架", NewContentTakenDownEvent(1, 2, "侵权"), contentenums.EventTypeTakenDown, "侵权"},
		{"恢复", NewContentRestoredEvent(1, 2), contentenums.EventTypeRestored, ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.evt.EventType)
			assert.Equal(t, int64(1), tt.evt.ContentID)
			assert.Equal(t, int64(2), tt.evt.AuthorID)
			assert.Equal(t, tt.reason, tt.evt.Reason)
		})
	}
}

// TestContentEventMarshalDropsSnapshotFields 通知式下发件箱 payload 不再携带状态快照
func TestContentEventMarshalDropsSnapshotFields(t *testing.T) {
	payload, err := NewContentPublishedEvent(1, 2).Marshal()
	require.NoError(t, err)
	assert.NotContains(t, payload, "visibility")
	assert.NotContains(t, payload, "content_type")
	assert.NotContains(t, payload, "published_at")

	evt, err := UnmarshalContentEvent(payload)
	require.NoError(t, err)
	assert.Equal(t, contentenums.EventTypePublished, evt.EventType)
	assert.Equal(t, int64(1), evt.ContentID)
	assert.Equal(t, int64(2), evt.AuthorID)
}
