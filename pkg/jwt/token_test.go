package jwt

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key"

func TestGenerateAndParseToken_Roundtrip(t *testing.T) {
	token, err := GenerateToken(42, time.Hour, testSecret)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := ParseToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, int64(42), claims.UserId)
}

func TestGenerateToken_ClaimsFields(t *testing.T) {
	before := time.Now().Add(-time.Second)
	token, err := GenerateToken(99, time.Hour, testSecret)
	require.NoError(t, err)

	claims, err := ParseToken(token, testSecret)
	require.NoError(t, err)

	assert.Equal(t, int64(99), claims.UserId)
	assert.Equal(t, "ran-feed", claims.Issuer)
	assert.Equal(t, "userToken", claims.Subject)
	assert.True(t, claims.ExpiresAt.After(before.Add(59*time.Minute)))
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, err := GenerateToken(1, time.Hour, testSecret)
	require.NoError(t, err)

	_, err = ParseToken(token, "wrong-secret")
	assert.Error(t, err)
}

func TestParseToken_ExpiredToken(t *testing.T) {
	token, err := GenerateToken(1, -time.Second, testSecret)
	require.NoError(t, err)

	_, err = ParseToken(token, testSecret)
	assert.Error(t, err)
}

func TestParseToken_GarbageString(t *testing.T) {
	_, err := ParseToken("not.a.jwt", testSecret)
	assert.Error(t, err)
}

func TestParseToken_EmptyString(t *testing.T) {
	_, err := ParseToken("", testSecret)
	assert.Error(t, err)
}

func TestParseToken_TamperedPayload(t *testing.T) {
	token, err := GenerateToken(1, time.Hour, testSecret)
	require.NoError(t, err)

	// 替换 payload 部分
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	parts[1] = "dGFtcGVyZWQ"
	tampered := strings.Join(parts, ".")

	_, err = ParseToken(tampered, testSecret)
	assert.Error(t, err)
}
