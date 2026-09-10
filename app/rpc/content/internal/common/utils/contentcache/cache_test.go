package contentcache

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/config"
	"ran-feed/app/rpc/content/internal/do"
)

// mockLoader 记录回源调用 只返回存在的 content_id
type mockLoader struct {
	calls   int
	lastIDs []int64
	details map[int64]*do.ContentDetailDO
	err     error
}

func (m *mockLoader) load(missIDs []int64) (map[int64]*do.ContentDetailDO, error) {
	m.calls++
	m.lastIDs = append([]int64(nil), missIDs...)
	if m.err != nil {
		return nil, m.err
	}
	res := make(map[int64]*do.ContentDetailDO, len(missIDs))
	for _, id := range missIDs {
		if d, ok := m.details[id]; ok {
			res[id] = d
		}
	}
	return res, nil
}

func newTestEnv(t *testing.T) (*redis.Redis, *miniredis.Miniredis, config.ContentCacheConfig) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	cfg := config.ContentCacheConfig{
		TTLSeconds:               600,
		NegativeTTLSeconds:       60,
		JitterMaxSeconds:         0, // 单测关 jitter 便于断言 TTL
		NegativeJitterMaxSeconds: 0,
	}
	return r, mr, cfg
}

func sampleDetail(id int64) *do.ContentDetailDO {
	return &do.ContentDetailDO{
		ContentID:   id,
		ContentType: int32(content.ContentType_CONTENT_TYPE_ARTICLE),
		AuthorID:    100 + id,
		Title:       "title",
		CoverURL:    "cover",
		PublishedAt: 1700000000,
	}
}

func TestBatchGet_AllHit(t *testing.T) {
	rds, _, cfg := newTestEnv(t)
	loader := &mockLoader{details: map[int64]*do.ContentDetailDO{1: sampleDetail(1), 2: sampleDetail(2)}}

	// 预热
	_, err := BatchGet(context.Background(), rds, cfg, []int64{1, 2}, loader.load)
	require.NoError(t, err)
	require.Equal(t, 1, loader.calls)

	// 二次全命中 不再回源
	res, err := BatchGet(context.Background(), rds, cfg, []int64{1, 2}, loader.load)
	require.NoError(t, err)
	assert.Equal(t, 1, loader.calls, "cache hit should not call loader")
	assert.Len(t, res, 2)
	assert.Equal(t, int64(101), res[1].AuthorID)
}

func TestBatchGet_PartialHit(t *testing.T) {
	rds, _, cfg := newTestEnv(t)
	loader := &mockLoader{details: map[int64]*do.ContentDetailDO{1: sampleDetail(1), 2: sampleDetail(2)}}

	// 只热 content 1
	_, err := BatchGet(context.Background(), rds, cfg, []int64{1}, loader.load)
	require.NoError(t, err)
	loader.calls = 0

	res, err := BatchGet(context.Background(), rds, cfg, []int64{1, 2}, loader.load)
	require.NoError(t, err)
	assert.Equal(t, 1, loader.calls)
	assert.Equal(t, []int64{2}, loader.lastIDs, "loader 只应收到 miss 的 id")
	assert.Len(t, res, 2)
}

func TestBatchGet_DedupAndSkipNonPositive(t *testing.T) {
	rds, _, cfg := newTestEnv(t)
	loader := &mockLoader{details: map[int64]*do.ContentDetailDO{1: sampleDetail(1), 2: sampleDetail(2), 3: sampleDetail(3)}}

	res, err := BatchGet(context.Background(), rds, cfg, []int64{1, 2, 3, 3, 0, -1}, loader.load)
	require.NoError(t, err)
	assert.Equal(t, 1, loader.calls)
	assert.Len(t, res, 3)

	gotIDs := make([]int64, 0, len(res))
	for id := range res {
		gotIDs = append(gotIDs, id)
	}
	slices.Sort(gotIDs)
	assert.Equal(t, []int64{1, 2, 3}, gotIDs)
}

func TestBatchGet_MissThenFill(t *testing.T) {
	rds, mr, cfg := newTestEnv(t)
	loader := &mockLoader{details: map[int64]*do.ContentDetailDO{42: sampleDetail(42)}}

	res, err := BatchGet(context.Background(), rds, cfg, []int64{42}, loader.load)
	require.NoError(t, err)
	require.NotNil(t, res[42])

	// 缓存写入正值
	raw, err := mr.Get(rediskey.BuildContentDetailKey(42))
	require.NoError(t, err)
	assert.NotEqual(t, rediskey.RedisContentDetailMissingSentinel, raw)
	var decoded do.ContentDetailDO
	require.NoError(t, json.Unmarshal([]byte(raw), &decoded))
	assert.Equal(t, int64(42), decoded.ContentID)

	ttl := mr.TTL(rediskey.BuildContentDetailKey(42))
	assert.InDelta(t, 600.0, ttl.Seconds(), 2.0)
}

func TestBatchGet_NotFoundWritesSentinel(t *testing.T) {
	rds, mr, cfg := newTestEnv(t)
	loader := &mockLoader{details: map[int64]*do.ContentDetailDO{}}

	res, err := BatchGet(context.Background(), rds, cfg, []int64{999}, loader.load)
	require.NoError(t, err)
	assert.Nil(t, res[999])

	// 负哨兵写入 negative TTL ~60s
	raw, err := mr.Get(rediskey.BuildContentDetailKey(999))
	require.NoError(t, err)
	assert.Equal(t, rediskey.RedisContentDetailMissingSentinel, raw)
	ttl := mr.TTL(rediskey.BuildContentDetailKey(999))
	assert.InDelta(t, 60.0, ttl.Seconds(), 2.0)
}

func TestBatchGet_SentinelHitSkipsLoader(t *testing.T) {
	rds, mr, cfg := newTestEnv(t)
	loader := &mockLoader{details: map[int64]*do.ContentDetailDO{1: sampleDetail(1)}}

	// content 2 预先种入负哨兵
	require.NoError(t, mr.Set(rediskey.BuildContentDetailKey(2), rediskey.RedisContentDetailMissingSentinel))

	res, err := BatchGet(context.Background(), rds, cfg, []int64{1, 2}, loader.load)
	require.NoError(t, err)
	assert.Equal(t, []int64{1}, loader.lastIDs, "命中哨兵的 id 不进 loader")
	assert.Len(t, res, 1)
	assert.NotNil(t, res[1])
	assert.Nil(t, res[2])
}

func TestBatchGet_CacheDisabledFallsThrough(t *testing.T) {
	rds, mr, _ := newTestEnv(t)
	cfg := config.ContentCacheConfig{TTLSeconds: 0} // 关闭
	loader := &mockLoader{details: map[int64]*do.ContentDetailDO{1: sampleDetail(1)}}

	res, err := BatchGet(context.Background(), rds, cfg, []int64{1}, loader.load)
	require.NoError(t, err)
	require.NotNil(t, res[1])
	assert.Equal(t, 1, loader.calls)
	// 关闭时不写缓存
	assert.False(t, mr.Exists(rediskey.BuildContentDetailKey(1)))
}

func TestBatchGet_LoaderErrorPropagates(t *testing.T) {
	rds, _, cfg := newTestEnv(t)
	loader := &mockLoader{err: errors.New("db down")}

	_, err := BatchGet(context.Background(), rds, cfg, []int64{1}, loader.load)
	require.Error(t, err)
}

func TestInvalidate_DeletesKeys(t *testing.T) {
	rds, mr, cfg := newTestEnv(t)
	loader := &mockLoader{details: map[int64]*do.ContentDetailDO{1: sampleDetail(1), 2: sampleDetail(2)}}

	_, err := BatchGet(context.Background(), rds, cfg, []int64{1, 2}, loader.load)
	require.NoError(t, err)
	require.True(t, mr.Exists(rediskey.BuildContentDetailKey(1)))

	require.NoError(t, Invalidate(context.Background(), rds, 1, 2, 0, -1))
	assert.False(t, mr.Exists(rediskey.BuildContentDetailKey(1)))
	assert.False(t, mr.Exists(rediskey.BuildContentDetailKey(2)))
}
