package im

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConversationCursor(t *testing.T) {
	encoded := encodeConversationCursor(1767225600000, 9900001)
	require.NotEmpty(t, encoded)

	at, id := decodeConversationCursor(encoded)
	assert.Equal(t, int64(1767225600000), at)
	assert.Equal(t, int64(9900001), id)

	for _, bad := range []string{"", "  ", "not-base64", encodeConversationCursor(0, 1), encodeConversationCursor(1, 0)} {
		at, id := decodeConversationCursor(bad)
		assert.Equal(t, int64(0), at, bad)
		assert.Equal(t, int64(0), id, bad)
	}
}
