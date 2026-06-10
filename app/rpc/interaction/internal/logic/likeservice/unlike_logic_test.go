package likeservicelogic

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/svc"
)

func newTestUnlikeLogic(t *testing.T) (*UnlikeLogic, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	ctx := context.Background()
	return &UnlikeLogic{
		ctx:    ctx,
		svcCtx: &svc.ServiceContext{Redis: r},
		Logger: logx.WithContext(ctx),
	}, mr
}

func unlikeReq(userID, contentID int64) *interaction.UnlikeReq {
	return &interaction.UnlikeReq{
		UserId:        userID,
		ContentId:     contentID,
		ContentUserId: 999,
		Scene:         interaction.Scene_ARTICLE,
	}
}

// TestUnlike_NoOpWhenNotLiked 未点赞直接取消应幂等无错误
func TestUnlike_NoOpWhenNotLiked(t *testing.T) {
	logic, _ := newTestUnlikeLogic(t)
	_, err := logic.Unlike(unlikeReq(1, 100))
	assert.NoError(t, err, "未点赞过的内容取消应无错误")
}

// TestUnlike_HappyPath 先 HSET 模拟点赞态 再取消应正常返回
func TestUnlike_HappyPath(t *testing.T) {
	logic, mr := newTestUnlikeLogic(t)
	userLikeKey := "like:user:1"
	mr.HSet(userLikeKey, "100", "1")

	_, err := logic.Unlike(unlikeReq(1, 100))
	require.NoError(t, err)

	val := mr.HGet(userLikeKey, "100")
	assert.Equal(t, "", val, "取消后 cid 字段应被删除")
}

// TestUnlike_Idempotent 重复取消应幂等
func TestUnlike_Idempotent(t *testing.T) {
	logic, mr := newTestUnlikeLogic(t)
	mr.HSet("like:user:1", "100", "1")

	_, err := logic.Unlike(unlikeReq(1, 100))
	require.NoError(t, err, "首次取消应成功")

	_, err = logic.Unlike(unlikeReq(1, 100))
	assert.NoError(t, err, "重复取消应幂等不报错")
}

// TestProcessUnlike_InvalidArgs 非法参数直接返回 false false nil 不打 Redis
func TestProcessUnlike_InvalidArgs(t *testing.T) {
	logic, _ := newTestUnlikeLogic(t)

	cases := []struct{ uid, cid int64 }{
		{0, 100},
		{1, 0},
		{-1, 100},
		{1, -1},
	}
	for _, c := range cases {
		changed, trusted, err := logic.processUnlike(c.uid, c.cid)
		assert.NoError(t, err)
		assert.False(t, changed)
		assert.False(t, trusted)
	}
}

// TestProcessUnlike_HotHitChanged 缓存中存在该 cid 时取消应 changed=true
func TestProcessUnlike_HotHitChanged(t *testing.T) {
	logic, mr := newTestUnlikeLogic(t)
	userLikeKey := "like:user:1"
	mr.HSet(userLikeKey, "100", "1")

	changed, _, err := logic.processUnlike(1, 100)
	require.NoError(t, err)
	assert.True(t, changed, "命中并删除 changed=true")
	assert.Equal(t, "", mr.HGet(userLikeKey, "100"), "字段已被删除")
}

// TestProcessUnlike_MissChangedFalse 缓存中无该 cid 时取消 changed=false
func TestProcessUnlike_MissChangedFalse(t *testing.T) {
	logic, _ := newTestUnlikeLogic(t)

	changed, _, err := logic.processUnlike(1, 100)
	require.NoError(t, err)
	assert.False(t, changed, "缓存里没该 cid changed=false")
}

// TestProcessUnlike_UntrustedWhenNotFull 残缺缓存 trusted 恒为 false
func TestProcessUnlike_UntrustedWhenNotFull(t *testing.T) {
	logic, mr := newTestUnlikeLogic(t)
	mr.HSet("like:user:1", "100", "1")

	_, trusted, err := logic.processUnlike(1, 100)
	require.NoError(t, err)
	assert.False(t, trusted, "无 _full 标记 缓存不可信")
}

// TestProcessUnlike_TrustedWhenFull 完整缓存且 cid 落在热区 trusted=true
func TestProcessUnlike_TrustedWhenFull(t *testing.T) {
	logic, mr := newTestUnlikeLogic(t)
	userLikeKey := "like:user:1"
	mr.HSet(userLikeKey, "_full", "1")
	mr.HSet(userLikeKey, "_mincid", "50")
	mr.HSet(userLikeKey, "100", "1")

	changed, trusted, err := logic.processUnlike(1, 100)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.True(t, trusted, "_full=1 且 cid >= mincid 可信")
}

// TestProcessUnlike_UntrustedWhenCold 完整缓存但 cid 在热区下方 视为冷数据
func TestProcessUnlike_UntrustedWhenCold(t *testing.T) {
	logic, mr := newTestUnlikeLogic(t)
	userLikeKey := "like:user:1"
	mr.HSet(userLikeKey, "_full", "1")
	mr.HSet(userLikeKey, "_mincid", "100")

	_, trusted, err := logic.processUnlike(1, 50)
	require.NoError(t, err)
	assert.False(t, trusted, "cid < mincid 冷数据 不可信")
}

// TestProcessUnlike_RecalcMinCidAfterDeleteMin 删除当前 mincid 后 mincid 应被重算为剩余最小
func TestProcessUnlike_RecalcMinCidAfterDeleteMin(t *testing.T) {
	logic, mr := newTestUnlikeLogic(t)
	userLikeKey := "like:user:1"
	mr.HSet(userLikeKey, "_full", "1")
	mr.HSet(userLikeKey, "_mincid", "100")
	mr.HSet(userLikeKey, "100", "1")
	mr.HSet(userLikeKey, "200", "1")
	mr.HSet(userLikeKey, "150", "1")

	changed, _, err := logic.processUnlike(1, 100)
	require.NoError(t, err)
	assert.True(t, changed)

	newMin := mr.HGet(userLikeKey, "_mincid")
	assert.Equal(t, "150", newMin, "删除原 mincid 后应重算为剩余业务字段的最小值")
}

// TestProcessUnlike_DeleteLastClearsMinCid 删除最后一个业务 cid 后 _mincid 应被 HDEL
// 元字段 _full 仍保留 故 key 不会被整体删除
func TestProcessUnlike_DeleteLastClearsMinCid(t *testing.T) {
	logic, mr := newTestUnlikeLogic(t)
	userLikeKey := "like:user:1"
	mr.HSet(userLikeKey, "_full", "1")
	mr.HSet(userLikeKey, "_mincid", "100")
	mr.HSet(userLikeKey, "100", "1")

	changed, _, err := logic.processUnlike(1, 100)
	require.NoError(t, err)
	assert.True(t, changed)
	assert.Equal(t, "", mr.HGet(userLikeKey, "_mincid"), "无剩余业务字段 _mincid 应被清除")
	assert.Equal(t, "1", mr.HGet(userLikeKey, "_full"), "_full 不受影响")
}
