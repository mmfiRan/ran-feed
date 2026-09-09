package likeservicelogic

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/do"
	"ran-feed/app/rpc/interaction/internal/entity/query"
	"ran-feed/app/rpc/interaction/internal/repositories"
	"ran-feed/app/rpc/interaction/internal/svc"
	"ran-feed/pkg/cache"
)

// fakeLikeRepo 测试桩 仅实现 batch 路径用到的 BatchIsLiked / QueryUserLikedTopN
// 其余接口方法保留 panic 提示误用
type fakeLikeRepo struct {
	topNByUser  map[int64][]int64 // user_id → topN cids（降序）
	likedByUser map[int64]map[int64]bool

	topNCalls  int32 // QueryUserLikedTopN 调用次数 用于验证单飞
	batchCalls int32 // BatchIsLiked 调用次数

	mu sync.Mutex
}

func (f *fakeLikeRepo) WithTx(_ *query.Query) repositories.LikeRepository { return f }
func (f *fakeLikeRepo) ApplyLike(_ *do.LikeDO) error                      { panic("not used") }
func (f *fakeLikeRepo) CancelLike(_ *do.LikeDO) error                     { panic("not used") }
func (f *fakeLikeRepo) BatchUpsert(_ []*do.LikeDO) error                  { panic("not used") }
func (f *fakeLikeRepo) GetByUserAndContent(_, _ int64) (*do.LikeDO, error) {
	panic("not used")
}
func (f *fakeLikeRepo) IsLiked(_, _ int64) (bool, error)         { panic("not used") }
func (f *fakeLikeRepo) GetLikedUserIDs(_ int64) ([]int64, error) { panic("not used") }

func (f *fakeLikeRepo) BatchIsLiked(userID int64, contentIDs []int64) (map[int64]bool, error) {
	atomic.AddInt32(&f.batchCalls, 1)
	f.mu.Lock()
	defer f.mu.Unlock()
	res := make(map[int64]bool, len(contentIDs))
	for _, cid := range contentIDs {
		if f.likedByUser[userID] != nil && f.likedByUser[userID][cid] {
			res[cid] = true
		}
	}
	return res, nil
}

func (f *fakeLikeRepo) QueryUserLikedTopN(userID int64, limit int) ([]int64, error) {
	atomic.AddInt32(&f.topNCalls, 1)
	f.mu.Lock()
	defer f.mu.Unlock()
	cids := f.topNByUser[userID]
	if len(cids) > limit {
		cids = cids[:limit]
	}
	out := make([]int64, len(cids))
	copy(out, cids)
	return out, nil
}

func newTestBatchLogic(t *testing.T, repo *fakeLikeRepo) (*BatchQueryIsLikedLogic, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	ctx := context.Background()
	return &BatchQueryIsLikedLogic{
		ctx: ctx,
		svcCtx: &svc.ServiceContext{
			Redis: r,
			// 测试用更激进的轮询参数 加速并发去重场景验证
			LikeUserRebuildLocker: cache.NewDistLocker(r,
				cache.WithWaitTimeout(2*time.Second),
				cache.WithPollInterval(10*time.Millisecond),
			),
		},
		Logger:   logx.WithContext(ctx),
		likeRepo: repo,
	}, mr
}

func batchReq(userID int64, cids ...int64) *interaction.BatchQueryIsLikedReq {
	uid := userID
	infos := make([]*interaction.LikeInfo, 0, len(cids))
	for _, cid := range cids {
		infos = append(infos, &interaction.LikeInfo{ContentId: cid})
	}
	return &interaction.BatchQueryIsLikedReq{
		UserId:    &uid,
		LikeInfos: infos,
	}
}

func likedMap(out *interaction.BatchQueryIsLikedRes) map[int64]bool {
	m := make(map[int64]bool, len(out.IsLikedInfos))
	for _, item := range out.IsLikedInfos {
		m[item.ContentId] = item.IsLiked
	}
	return m
}

// TestBatch_EmptyRequest 空请求直接返回空结果
func TestBatch_EmptyRequest(t *testing.T) {
	logic, _ := newTestBatchLogic(t, &fakeLikeRepo{})
	out, err := logic.BatchQueryIsLiked(&interaction.BatchQueryIsLikedReq{})
	require.NoError(t, err)
	assert.Empty(t, out.IsLikedInfos)
}

// TestBatch_NoUserID UserId 为 nil 时返回全 false
func TestBatch_NoUserID(t *testing.T) {
	logic, _ := newTestBatchLogic(t, &fakeLikeRepo{})
	out, err := logic.BatchQueryIsLiked(&interaction.BatchQueryIsLikedReq{
		LikeInfos: []*interaction.LikeInfo{{ContentId: 1}, {ContentId: 2}},
	})
	require.NoError(t, err)
	for _, item := range out.IsLikedInfos {
		assert.False(t, item.IsLiked)
	}
}

// TestBatch_CacheMissTriggersRebuild 缓存不存在时触发重建 重建后按 topN 填充结果
func TestBatch_CacheMissTriggersRebuild(t *testing.T) {
	repo := &fakeLikeRepo{
		topNByUser: map[int64][]int64{
			1: {300, 200, 100}, // 降序
		},
	}
	logic, mr := newTestBatchLogic(t, repo)

	out, err := logic.BatchQueryIsLiked(batchReq(1, 100, 200, 999))
	require.NoError(t, err)
	m := likedMap(out)
	assert.True(t, m[100], "100 在 topN 中应为 true")
	assert.True(t, m[200], "200 在 topN 中应为 true")
	assert.False(t, m[999], "999 不在 topN 中应为 false")

	assert.Equal(t, int32(1), atomic.LoadInt32(&repo.topNCalls), "重建应只调一次 DB topN")
	assert.Equal(t, int32(0), atomic.LoadInt32(&repo.batchCalls), "999 落在热区 不应触发 DB 兜底")

	userLikeKey := "like:user:1"
	assert.Equal(t, "1", mr.HGet(userLikeKey, "_full"), "重建后 _full=1")
	assert.Equal(t, "100", mr.HGet(userLikeKey, "_mincid"), "_mincid 应为 topN 末尾值")
}

// TestBatch_CacheHitFullReturnsCacheOnly _full=1 缓存可信 命中信缓存 未命中=false
func TestBatch_CacheHitFullReturnsCacheOnly(t *testing.T) {
	repo := &fakeLikeRepo{}
	logic, mr := newTestBatchLogic(t, repo)

	userLikeKey := "like:user:1"
	mr.HSet(userLikeKey, "_full", "1")
	mr.HSet(userLikeKey, "_mincid", "100")
	mr.HSet(userLikeKey, "100", "1")
	mr.HSet(userLikeKey, "200", "1")

	out, err := logic.BatchQueryIsLiked(batchReq(1, 100, 200, 999))
	require.NoError(t, err)
	m := likedMap(out)
	assert.True(t, m[100])
	assert.True(t, m[200])
	assert.False(t, m[999], "999 >= mincid 且不在缓存 视为未点赞")

	assert.Equal(t, int32(0), atomic.LoadInt32(&repo.topNCalls), "可信缓存不应重建")
	assert.Equal(t, int32(0), atomic.LoadInt32(&repo.batchCalls), "999 在热区 不走 DB")
}

// TestBatch_ColdDataFallsBackToDB cid < minCID 视为冷数据回源 DB
func TestBatch_ColdDataFallsBackToDB(t *testing.T) {
	repo := &fakeLikeRepo{
		likedByUser: map[int64]map[int64]bool{
			1: {50: true},
		},
	}
	logic, mr := newTestBatchLogic(t, repo)

	userLikeKey := "like:user:1"
	mr.HSet(userLikeKey, "_full", "1")
	mr.HSet(userLikeKey, "_mincid", "100")
	mr.HSet(userLikeKey, "100", "1")

	out, err := logic.BatchQueryIsLiked(batchReq(1, 50, 100))
	require.NoError(t, err)
	m := likedMap(out)
	assert.True(t, m[50], "50 是冷数据 走 DB 返回 true")
	assert.True(t, m[100], "100 在热区命中")

	assert.Equal(t, int32(1), atomic.LoadInt32(&repo.batchCalls), "应触发一次冷数据 DB 查询")
	assert.Equal(t, int32(0), atomic.LoadInt32(&repo.topNCalls), "_full=1 不该重建")
}

// TestBatch_StaleCacheTriggersRebuild _full=0 视为残缺 触发重建
func TestBatch_StaleCacheTriggersRebuild(t *testing.T) {
	repo := &fakeLikeRepo{
		topNByUser: map[int64][]int64{
			1: {500, 400},
		},
	}
	logic, mr := newTestBatchLogic(t, repo)

	userLikeKey := "like:user:1"
	mr.HSet(userLikeKey, "300", "1") // 残缺缓存 无 _full

	out, err := logic.BatchQueryIsLiked(batchReq(1, 400, 500))
	require.NoError(t, err)
	m := likedMap(out)
	assert.True(t, m[400])
	assert.True(t, m[500])
	assert.Equal(t, int32(1), atomic.LoadInt32(&repo.topNCalls))
	assert.Equal(t, "1", mr.HGet(userLikeKey, "_full"))
	assert.Equal(t, "", mr.HGet(userLikeKey, "300"), "残缺脏字段应被重建清除")
}

// TestBatch_EmptyTopNStillMarksFull 用户无任何点赞 也应写空 _full=1 hash 避免反复重建
func TestBatch_EmptyTopNStillMarksFull(t *testing.T) {
	repo := &fakeLikeRepo{topNByUser: map[int64][]int64{}}
	logic, mr := newTestBatchLogic(t, repo)

	out, err := logic.BatchQueryIsLiked(batchReq(1, 100, 200))
	require.NoError(t, err)
	for _, item := range out.IsLikedInfos {
		assert.False(t, item.IsLiked)
	}
	assert.Equal(t, "1", mr.HGet("like:user:1", "_full"), "无点赞用户也应标 _full=1")
}

// TestBatch_DistLockDedupRebuild 同 user 并发 N 个请求只触发一次 DB 重建
// 分布式锁保证同 lockKey 全集群只一个 rebuild 其余轮询 cache 等结果
func TestBatch_DistLockDedupRebuild(t *testing.T) {
	repo := &fakeLikeRepo{
		topNByUser: map[int64][]int64{
			1: {100, 50},
		},
	}
	logic, _ := newTestBatchLogic(t, repo)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			<-start
			_, err := logic.BatchQueryIsLiked(batchReq(1, 100))
			assert.NoError(t, err)
		}()
	}
	close(start)
	wg.Wait()

	calls := atomic.LoadInt32(&repo.topNCalls)
	assert.LessOrEqual(t, calls, int32(2), "50 并发应被单飞合并 DB 调用次数不超过 2")
}

// TestBatch_RedisErrorDegrades Redis 整体不可用时降级走 DB
func TestBatch_RedisErrorDegrades(t *testing.T) {
	repo := &fakeLikeRepo{
		likedByUser: map[int64]map[int64]bool{
			1: {100: true},
		},
	}
	logic, mr := newTestBatchLogic(t, repo)
	mr.SetError("redis offline") // miniredis 模拟全局错误

	out, err := logic.BatchQueryIsLiked(batchReq(1, 100, 200))
	require.NoError(t, err)
	m := likedMap(out)
	assert.True(t, m[100], "降级 DB 后 100 仍然返回 true")
	assert.False(t, m[200])
	assert.GreaterOrEqual(t, atomic.LoadInt32(&repo.batchCalls), int32(1))
}

// TestBatch_DedupContentIDs 入参重复 cid 结果一致 不重复打 cache
func TestBatch_DedupContentIDs(t *testing.T) {
	repo := &fakeLikeRepo{topNByUser: map[int64][]int64{1: {100}}}
	logic, _ := newTestBatchLogic(t, repo)

	out, err := logic.BatchQueryIsLiked(batchReq(1, 100, 100, 100))
	require.NoError(t, err)
	require.Len(t, out.IsLikedInfos, 3, "返回顺序与入参一致 保留重复项")
	for _, item := range out.IsLikedInfos {
		assert.Equal(t, int64(100), item.ContentId)
		assert.True(t, item.IsLiked)
	}
}

// TestBatch_LockBusyFallsBackToDB 模拟另一个实例长期持锁 本实例等待超时后降级走 DB
// 体现"拿不到锁→ErrLockBusy→fillAllFromDB"的兜底链路
func TestBatch_LockBusyFallsBackToDB(t *testing.T) {
	repo := &fakeLikeRepo{
		likedByUser: map[int64]map[int64]bool{
			1: {100: true},
		},
	}
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	// 极短超时让本测试快速触发 ErrLockBusy
	locker := cache.NewDistLocker(r,
		cache.WithWaitTimeout(100*time.Millisecond),
		cache.WithPollInterval(20*time.Millisecond),
	)
	logic := &BatchQueryIsLikedLogic{
		ctx: context.Background(),
		svcCtx: &svc.ServiceContext{
			Redis:                 r,
			LikeUserRebuildLocker: locker,
		},
		Logger:   logx.WithContext(context.Background()),
		likeRepo: repo,
	}

	// 模拟另一个实例已持有锁 长期不释放
	require.NoError(t, mr.Set(cache.BuildLockKey("like:user:1"), "other-instance-owner"))
	mr.SetTTL(cache.BuildLockKey("like:user:1"), 30*time.Second)

	out, err := logic.BatchQueryIsLiked(batchReq(1, 100, 200))
	require.NoError(t, err, "锁拿不到也不应向上抛错 而是走 DB 兜底")

	m := likedMap(out)
	assert.True(t, m[100], "DB 兜底应返回 100 已点赞")
	assert.False(t, m[200])
	assert.GreaterOrEqual(t, atomic.LoadInt32(&repo.batchCalls), int32(1), "应触发至少一次 DB 兜底")
	assert.Equal(t, int32(0), atomic.LoadInt32(&repo.topNCalls), "未拿到锁不应执行重建 topN")
}

// TestBatch_InvalidContentIDsSkipped 非法 cid (≤0) 跳过 不影响其他 cid
func TestBatch_InvalidContentIDsSkipped(t *testing.T) {
	repo := &fakeLikeRepo{topNByUser: map[int64][]int64{1: {100}}}
	logic, _ := newTestBatchLogic(t, repo)

	out, err := logic.BatchQueryIsLiked(batchReq(1, -1, 0, 100))
	require.NoError(t, err)
	m := likedMap(out)
	assert.True(t, m[100])
	assert.False(t, m[-1])
	assert.False(t, m[0])
}
