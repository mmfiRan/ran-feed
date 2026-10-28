package groupservicelogic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeMemberIDs(t *testing.T) {
	got, err := sanitizeMemberIDs(1, []int64{1, 2, 2, 3})
	require.NoError(t, err)
	assert.Equal(t, []int64{2, 3}, got)

	got, err = sanitizeMemberIDs(1, nil)
	require.NoError(t, err)
	assert.Nil(t, got)

	_, err = sanitizeMemberIDs(1, []int64{0})
	assert.Error(t, err)
	_, err = sanitizeMemberIDs(1, []int64{-1})
	assert.Error(t, err)
}

func TestValidateGroupMeta(t *testing.T) {
	name, err := validateGroupMeta("  测试群  ", "a.png")
	require.NoError(t, err)
	assert.Equal(t, "测试群", name)

	_, err = validateGroupMeta("   ", "")
	assert.Error(t, err)

	_, err = validateGroupMeta(string(make([]byte, 65)), "")
	assert.Error(t, err)
}
