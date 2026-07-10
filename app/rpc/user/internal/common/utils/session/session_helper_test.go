package session

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/zeromicro/go-zero/core/stores/redis"

	rediskey "ran-feed/app/rpc/user/internal/common/consts/redis"
)

func TestRemoveByUserID(t *testing.T) {
	mr := miniredis.RunT(t)
	r := redis.New(mr.Addr())
	ctx := context.Background()

	t.Run("有会话则删除双向key", func(t *testing.T) {
		userID := int64(123)
		token := "test-token-123"

		// 预设双向 key
		tokenKey := rediskey.BuildUserSessionKey(token)
		userKey := rediskey.BuildUserSessionUserKey(userID)
		mr.Set(tokenKey, "123")
		mr.Set(userKey, token)

		err := RemoveByUserID(ctx, r, userID)
		assert.NoError(t, err)

		// 验证双向 key 都被删除
		exists, _ := r.ExistsCtx(ctx, tokenKey)
		assert.False(t, exists)
		exists, _ = r.ExistsCtx(ctx, userKey)
		assert.False(t, exists)
	})

	t.Run("无会话则no-op", func(t *testing.T) {
		userID := int64(999)

		// 反向索引不存在 GetCtx 返回空串不报错
		err := RemoveByUserID(ctx, r, userID)
		assert.NoError(t, err) // 空串走 no-op 分支
	})

	t.Run("反向索引为空串则no-op", func(t *testing.T) {
		userID := int64(888)
		userKey := rediskey.BuildUserSessionUserKey(userID)
		mr.Set(userKey, "")

		err := RemoveByUserID(ctx, r, userID)
		assert.NoError(t, err)

		// 空串不调用 RemoveSession，反向索引仍在
		exists, _ := r.ExistsCtx(ctx, userKey)
		assert.True(t, exists)
	})
}

func TestSaveAndRemoveSession(t *testing.T) {
	mr := miniredis.RunT(t)
	r := redis.New(mr.Addr())
	ctx := context.Background()

	userID := int64(100)
	token := "token-abc"
	ttl := 10 * time.Second

	// Save
	err := SaveSession(ctx, r, userID, token, ttl)
	assert.NoError(t, err)

	tokenKey := rediskey.BuildUserSessionKey(token)
	userKey := rediskey.BuildUserSessionUserKey(userID)

	val, _ := r.GetCtx(ctx, tokenKey)
	assert.Equal(t, "100", val)
	val, _ = r.GetCtx(ctx, userKey)
	assert.Equal(t, token, val)

	// Remove
	err = RemoveSession(ctx, r, userID, token)
	assert.NoError(t, err)

	exists, _ := r.ExistsCtx(ctx, tokenKey)
	assert.False(t, exists)
	exists, _ = r.ExistsCtx(ctx, userKey)
	assert.False(t, exists)
}