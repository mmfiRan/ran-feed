package feedservicelogic

import (
	"context"
	"testing"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/svc"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

func newTestFollowLogic(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *FollowFeedLogic) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	return mr, r, &FollowFeedLogic{
		ctx:    context.Background(),
		svcCtx: &svc.ServiceContext{Redis: r},
		Logger: logx.WithContext(context.Background()),
	}
}

// TestPickBigVFollowees_OnlyGlobalMembers 只返回落在全局大 V 集合中的候选
func TestPickBigVFollowees_OnlyGlobalMembers(t *testing.T) {
	_, r, l := newTestFollowLogic(t)
	ctx := context.Background()

	_, err := r.SaddCtx(ctx, rediskey.RedisFeedBigVGlobalKey, "100", "200")
	require.NoError(t, err)

	got, err := l.pickBigVFollowees(ctx, []int64{100, 300, 200, -1, 0})
	require.NoError(t, err)
	assert.ElementsMatch(t, []int64{100, 200}, got)
}

// TestPickBigVFollowees_EmptyCandidates 空候选直接返回不触达 Redis
func TestPickBigVFollowees_EmptyCandidates(t *testing.T) {
	_, _, l := newTestFollowLogic(t)

	got, err := l.pickBigVFollowees(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestExcludeFollowees 求差集保序 空排除集原样返回
func TestExcludeFollowees(t *testing.T) {
	assert.Equal(t, []int64{2, 4}, excludeFollowees([]int64{1, 2, 3, 4}, []int64{1, 3}))
	assert.Equal(t, []int64{1, 2}, excludeFollowees([]int64{1, 2}, nil))
	assert.Empty(t, excludeFollowees(nil, []int64{1}))
}

// TestWriteAndReadPullAuthors 拉模式集读写一致 空集写哨兵后读回为空且 key 存在防穿透
func TestWriteAndReadPullAuthors(t *testing.T) {
	mr, _, l := newTestFollowLogic(t)
	ctx := context.Background()
	key := rediskey.BuildFollowPullKey(999)

	require.NoError(t, l.writePullAuthors(ctx, key, []int64{7, 8}, 300))
	assert.ElementsMatch(t, []int64{7, 8}, l.readPullAuthors(ctx, key))

	require.NoError(t, l.writePullAuthors(ctx, key, nil, 300))
	assert.Empty(t, l.readPullAuthors(ctx, key))
	assert.True(t, mr.Exists(key), "空集写哨兵 保证 key 存在不再反复重建")
}
