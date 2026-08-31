package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashPasswordRoundtrip(t *testing.T) {
	hash, err := HashPassword("Passw0rd")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	assert.True(t, CheckPassword(hash, "Passw0rd"))
	assert.False(t, CheckPassword(hash, "wrong"))
}

// TestHashPasswordUniqueSalt 同一密码两次哈希结果不同 bcrypt 内嵌随机盐
func TestHashPasswordUniqueSalt(t *testing.T) {
	h1, err := HashPassword("Passw0rd")
	require.NoError(t, err)
	h2, err := HashPassword("Passw0rd")
	require.NoError(t, err)

	assert.NotEqual(t, h1, h2)
	assert.True(t, CheckPassword(h1, "Passw0rd"))
	assert.True(t, CheckPassword(h2, "Passw0rd"))
}
