package usercache

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

	rediskey "ran-feed/app/rpc/user/internal/common/consts/redis"
	"ran-feed/app/rpc/user/internal/config"
	"ran-feed/app/rpc/user/internal/do"
	"ran-feed/app/rpc/user/internal/entity/model"
	"ran-feed/app/rpc/user/internal/entity/query"
	"ran-feed/app/rpc/user/internal/repositories"
)

type mockRepo struct {
	getByIDCalls       int
	batchGetByIDsCalls int
	users              map[int64]*do.UserDO
	batchErr           error
	getErr             error
}

func (m *mockRepo) WithTx(_ *query.Query) repositories.UserRepository { return m }
func (m *mockRepo) GetByMobile(string) (*do.UserDO, error)            { return nil, nil }
func (m *mockRepo) Create(*do.UserDO) (int64, error)                  { return 0, nil }
func (m *mockRepo) BatchGetActiveForIndex([]int64) (map[int64]*model.RanFeedUser, error) {
	return nil, nil
}
func (m *mockRepo) ScanActiveForIndex(int64, int) ([]*model.RanFeedUser, error) { return nil, nil }
func (m *mockRepo) AdminPageUsers(int32, string, int, int) ([]*model.RanFeedUser, int64, error) {
	return nil, 0, nil
}
func (m *mockRepo) AdminGetByID(int64) (*model.RanFeedUser, error)       { return nil, nil }
func (m *mockRepo) AdminUpdateStatus(int64, int32, int64) (int64, error) { return 0, nil }

func (m *mockRepo) GetByID(userID int64) (*do.UserDO, error) {
	m.getByIDCalls++
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.users[userID], nil
}

func (m *mockRepo) BatchGetByIDs(userIDs []int64) (map[int64]*do.UserDO, error) {
	m.batchGetByIDsCalls++
	if m.batchErr != nil {
		return nil, m.batchErr
	}
	res := make(map[int64]*do.UserDO, len(userIDs))
	for _, id := range userIDs {
		if u, ok := m.users[id]; ok {
			res[id] = u
		}
	}
	return res, nil
}

func newTestEnv(t *testing.T) (*redis.Redis, *miniredis.Miniredis, config.UserCacheConfig) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	cfg := config.UserCacheConfig{
		TTLSeconds:               600,
		NegativeTTLSeconds:       60,
		JitterMaxSeconds:         0, // 单测关 jitter，便于断言 TTL
		NegativeJitterMaxSeconds: 0,
	}
	return r, mr, cfg
}

func sampleUser(id int64) *do.UserDO {
	return &do.UserDO{
		ID:       id,
		Username: "u",
		Nickname: "nick",
		Avatar:   "avatar",
		Bio:      "bio",
		Mobile:   "13800000000",
		Gender:   1,
		Status:   10,
	}
}

// ---------- 单查 ----------

func TestGet_HitCache(t *testing.T) {
	rds, mr, cfg := newTestEnv(t)
	repo := &mockRepo{users: map[int64]*do.UserDO{1: sampleUser(1)}}

	// 预先把缓存写好
	first, err := Get(context.Background(), rds, repo, cfg, 1)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, 1, repo.getByIDCalls)
	assert.True(t, mr.Exists(rediskey.BuildUserInfoKey(1)))

	// 第二次应当命中缓存，不再调 DB
	second, err := Get(context.Background(), rds, repo, cfg, 1)
	require.NoError(t, err)
	require.NotNil(t, second)
	assert.Equal(t, 1, repo.getByIDCalls, "cache hit should not hit DB")
	assert.Equal(t, "nick", second.Nickname)
}

func TestGet_MissThenFill(t *testing.T) {
	rds, mr, cfg := newTestEnv(t)
	repo := &mockRepo{users: map[int64]*do.UserDO{42: sampleUser(42)}}

	u, err := Get(context.Background(), rds, repo, cfg, 42)
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, int64(42), u.ID)

	// 缓存应被写入正值
	raw, err := mr.Get(rediskey.BuildUserInfoKey(42))
	require.NoError(t, err)
	assert.NotEqual(t, rediskey.RedisUserInfoMissingSentinel, raw)
	var decoded userCacheDO
	require.NoError(t, json.Unmarshal([]byte(raw), &decoded))
	assert.Equal(t, int64(42), decoded.ID)

	// TTL 应当落在 [TTL, TTL+Jitter] 内（这里关了 jitter）
	ttl := mr.TTL(rediskey.BuildUserInfoKey(42))
	assert.InDelta(t, 600.0, ttl.Seconds(), 2.0)
}

func TestGet_NotFoundWritesSentinel(t *testing.T) {
	rds, mr, cfg := newTestEnv(t)
	repo := &mockRepo{users: map[int64]*do.UserDO{}}

	u, err := Get(context.Background(), rds, repo, cfg, 999)
	require.NoError(t, err)
	assert.Nil(t, u)

	// 哨兵被写入
	raw, err := mr.Get(rediskey.BuildUserInfoKey(999))
	require.NoError(t, err)
	assert.Equal(t, rediskey.RedisUserInfoMissingSentinel, raw)

	// negative TTL ~60s
	ttl := mr.TTL(rediskey.BuildUserInfoKey(999))
	assert.InDelta(t, 60.0, ttl.Seconds(), 2.0)
}

func TestGet_SentinelHitSkipsDB(t *testing.T) {
	rds, _, cfg := newTestEnv(t)
	repo := &mockRepo{users: map[int64]*do.UserDO{}}

	// 第一次：DB miss → 写哨兵
	_, err := Get(context.Background(), rds, repo, cfg, 7)
	require.NoError(t, err)
	require.Equal(t, 1, repo.getByIDCalls)

	// 第二次：命中哨兵，不再打 DB
	u, err := Get(context.Background(), rds, repo, cfg, 7)
	require.NoError(t, err)
	assert.Nil(t, u)
	assert.Equal(t, 1, repo.getByIDCalls, "sentinel hit should not trigger DB")
}

// ---------- 批查 ----------

func TestBatchGet_AllHit(t *testing.T) {
	rds, _, cfg := newTestEnv(t)
	repo := &mockRepo{users: map[int64]*do.UserDO{
		1: sampleUser(1),
		2: sampleUser(2),
	}}

	// 预热
	_, err := BatchGet(context.Background(), rds, repo, cfg, []int64{1, 2})
	require.NoError(t, err)
	require.Equal(t, 1, repo.batchGetByIDsCalls)

	// 二次：全命中
	res, err := BatchGet(context.Background(), rds, repo, cfg, []int64{1, 2})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.batchGetByIDsCalls)
	assert.Len(t, res, 2)
	assert.NotNil(t, res[1])
	assert.NotNil(t, res[2])
}

func TestBatchGet_PartialHit(t *testing.T) {
	rds, _, cfg := newTestEnv(t)
	repo := &mockRepo{users: map[int64]*do.UserDO{
		1: sampleUser(1),
		2: sampleUser(2),
	}}

	// 预热只热 user 1
	_, err := Get(context.Background(), rds, repo, cfg, 1)
	require.NoError(t, err)
	repo.getByIDCalls = 0

	res, err := BatchGet(context.Background(), rds, repo, cfg, []int64{1, 2})
	require.NoError(t, err)
	// 只应当查一次 DB（针对 miss 的 user 2）
	assert.Equal(t, 1, repo.batchGetByIDsCalls)
	assert.Len(t, res, 2)
}

func TestBatchGet_AllMiss(t *testing.T) {
	rds, _, cfg := newTestEnv(t)
	repo := &mockRepo{users: map[int64]*do.UserDO{
		1: sampleUser(1),
		2: sampleUser(2),
		3: sampleUser(3),
	}}

	res, err := BatchGet(context.Background(), rds, repo, cfg, []int64{1, 2, 3, 3, 0, -1})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.batchGetByIDsCalls)
	assert.Len(t, res, 3)

	gotIDs := make([]int64, 0, len(res))
	for id := range res {
		gotIDs = append(gotIDs, id)
	}
	slices.Sort(gotIDs)
	assert.Equal(t, []int64{1, 2, 3}, gotIDs)
}

func TestBatchGet_WithSentinel(t *testing.T) {
	rds, mr, cfg := newTestEnv(t)
	repo := &mockRepo{users: map[int64]*do.UserDO{1: sampleUser(1)}}

	// user 2 不存在，预先种入哨兵
	require.NoError(t, mr.Set(rediskey.BuildUserInfoKey(2), rediskey.RedisUserInfoMissingSentinel))

	res, err := BatchGet(context.Background(), rds, repo, cfg, []int64{1, 2})
	require.NoError(t, err)
	// user 1 走 DB 一次（首次 miss），user 2 命中哨兵不打 DB
	assert.Equal(t, 1, repo.batchGetByIDsCalls)
	assert.Len(t, res, 1)
	assert.NotNil(t, res[1])
	assert.Nil(t, res[2])
}

// ---------- 边界 / 异常 ----------

func TestGet_CacheDisabledFallsThrough(t *testing.T) {
	rds, mr, _ := newTestEnv(t)
	cfg := config.UserCacheConfig{TTLSeconds: 0} // 关闭
	repo := &mockRepo{users: map[int64]*do.UserDO{1: sampleUser(1)}}

	u, err := Get(context.Background(), rds, repo, cfg, 1)
	require.NoError(t, err)
	require.NotNil(t, u)
	// 缓存应当为空（关闭时不写）
	assert.False(t, mr.Exists(rediskey.BuildUserInfoKey(1)))
	assert.Equal(t, 1, repo.getByIDCalls)
}

func TestGet_DBErrorPropagates(t *testing.T) {
	rds, _, cfg := newTestEnv(t)
	repo := &mockRepo{getErr: errors.New("db down")}

	_, err := Get(context.Background(), rds, repo, cfg, 1)
	require.Error(t, err)
}
