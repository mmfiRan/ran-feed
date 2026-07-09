package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSaltAndHashRoundtrip(t *testing.T) {
	salt, err := GenerateSalt(16)
	require.NoError(t, err)
	assert.Len(t, salt, 32) // 16 字节十六进制

	salt2, err := GenerateSalt(16)
	require.NoError(t, err)
	assert.NotEqual(t, salt, salt2) // 两次盐不同

	hash, err := HashPassword("Passw0rd" + salt)
	require.NoError(t, err)
	assert.True(t, CheckPassword(hash, "Passw0rd"+salt))
	assert.False(t, CheckPassword(hash, "wrong"+salt))
}
