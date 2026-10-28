package consts

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleConversationBizKey(t *testing.T) {
	assert.Equal(t, "S:100_200", SingleConversationBizKey(100, 200))
	assert.Equal(t, "S:100_200", SingleConversationBizKey(200, 100))

	a, b, ok := ParseSingleConversationBizKey("S:100_200")
	require.True(t, ok)
	assert.Equal(t, int64(100), a)
	assert.Equal(t, int64(200), b)

	for _, bad := range []string{"", "G:1", "S:", "S:abc_200", "S:0_200", "S:100", "S:100_200_300"} {
		_, _, ok := ParseSingleConversationBizKey(bad)
		assert.False(t, ok, bad)
	}
}

func TestGroupConversationBizKey(t *testing.T) {
	assert.Equal(t, "G:42", GroupConversationBizKey(42))
}
