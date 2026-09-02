package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"ran-feed/app/admin/internal/common/consts"
)

func newTestRedis(t *testing.T) (*redis.Redis, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	return redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType}), mr
}

func TestHasPermission(t *testing.T) {
	perms := map[string]struct{}{"content:list": {}, "user:ban": {}}
	tests := []struct {
		name     string
		required string
		want     bool
	}{
		{name: "命中", required: "content:list", want: true},
		{name: "未命中", required: "content:takedown", want: false},
		{name: "空 required 不放行", required: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, HasPermission(perms, tt.required))
		})
	}
}

func TestToSetExcludesSentinel(t *testing.T) {
	set := toSet([]string{consts.RedisAdminPermLoadedSentinel, "content:list", "user:ban"})
	_, hasSentinel := set[consts.RedisAdminPermLoadedSentinel]
	assert.False(t, hasSentinel)
	assert.Len(t, set, 2)
	assert.Contains(t, set, "content:list")
}

func TestLoadPermissions_MissThenHit(t *testing.T) {
	r, _ := newTestRedis(t)
	calls := 0
	loader := func(_ context.Context, _ int64) ([]string, error) {
		calls++
		return []string{"content:list", "user:ban"}, nil
	}

	// 首次未命中回源并缓存
	set, err := LoadPermissions(context.Background(), r, 1, 600, loader)
	require.NoError(t, err)
	assert.Len(t, set, 2)
	assert.Equal(t, 1, calls)

	// 二次命中缓存不再回源
	set2, err := LoadPermissions(context.Background(), r, 1, 600, loader)
	require.NoError(t, err)
	assert.Len(t, set2, 2)
	assert.Equal(t, 1, calls)
}

func TestLoadPermissions_ZeroPermCachedBySentinel(t *testing.T) {
	r, _ := newTestRedis(t)
	calls := 0
	loader := func(_ context.Context, _ int64) ([]string, error) {
		calls++
		return nil, nil
	}

	set, err := LoadPermissions(context.Background(), r, 2, 600, loader)
	require.NoError(t, err)
	assert.Empty(t, set)
	assert.Equal(t, 1, calls)

	// 零权限也被哨兵缓存 二次不回源
	_, err = LoadPermissions(context.Background(), r, 2, 600, loader)
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestLoadPermissions_InvalidateReloads(t *testing.T) {
	r, _ := newTestRedis(t)
	calls := 0
	loader := func(_ context.Context, _ int64) ([]string, error) {
		calls++
		return []string{"content:list"}, nil
	}

	_, err := LoadPermissions(context.Background(), r, 3, 600, loader)
	require.NoError(t, err)
	require.Equal(t, 1, calls)

	// 外部失效(角色/权限变更后由 admin-rpc 删缓存)后重新回源
	_, err = r.DelCtx(context.Background(), consts.BuildAdminPermKey(3))
	require.NoError(t, err)

	// 失效后重新回源
	_, err = LoadPermissions(context.Background(), r, 3, 600, loader)
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestLoadPermissions_LoaderError(t *testing.T) {
	r, _ := newTestRedis(t)
	loader := func(_ context.Context, _ int64) ([]string, error) {
		return nil, errors.New("rpc down")
	}
	_, err := LoadPermissions(context.Background(), r, 4, 600, loader)
	assert.Error(t, err)
}
