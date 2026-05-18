package likeservicelogic

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"ran-feed/app/rpc/interaction/interaction"
	"ran-feed/app/rpc/interaction/internal/svc"
)

func TestMain(m *testing.M) {
	logx.Disable()
	os.Exit(m.Run())
}

func newTestLikeLogic(t *testing.T) (*LikeLogic, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	ctx := context.Background()
	return &LikeLogic{
		ctx:    ctx,
		svcCtx: &svc.ServiceContext{Redis: r},
		Logger: logx.WithContext(ctx),
	}, mr
}

func likeReq(userID, contentID int64) *interaction.LikeReq {
	return &interaction.LikeReq{
		UserId:        userID,
		ContentId:     contentID,
		ContentUserId: 999,
		Scene:         interaction.Scene_ARTICLE,
	}
}

// TestLike_FirstLikeSucceeds 首次点赞不报错
func TestLike_FirstLikeSucceeds(t *testing.T) {
	logic, _ := newTestLikeLogic(t)
	_, err := logic.Like(likeReq(1, 100))
	assert.NoError(t, err)
}

// TestLike_Idempotent 同用户同内容重复点赞两次，均无错误（幂等）
func TestLike_Idempotent(t *testing.T) {
	logic, _ := newTestLikeLogic(t)
	req := likeReq(1, 100)

	_, err := logic.Like(req)
	require.NoError(t, err, "第一次点赞应成功")

	_, err = logic.Like(req)
	assert.NoError(t, err, "重复点赞应幂等，不报错")
}

// TestLike_IdempotentVerifyHash 验证 Redis Hash 中点赞只记录一次
func TestLike_IdempotentVerifyHash(t *testing.T) {
	logic, mr := newTestLikeLogic(t)
	req := likeReq(2, 200)

	_, err := logic.Like(req)
	require.NoError(t, err)

	_, err = logic.Like(req)
	require.NoError(t, err)

	// 用户维度 Hash：content_id 作为 field，值应为 "1"，且只存在一次
	userLikeKey := "like:user:" + strconv.FormatInt(req.UserId, 10)
	val := mr.HGet(userLikeKey, strconv.FormatInt(req.ContentId, 10))
	assert.Equal(t, "1", val, "点赞值应为 1，且 Hash 中只存在一次")
}

// TestLike_DifferentUsersIndependent 不同用户点赞同一内容互不影响
func TestLike_DifferentUsersIndependent(t *testing.T) {
	logic, _ := newTestLikeLogic(t)

	_, err := logic.Like(likeReq(1, 300))
	require.NoError(t, err)

	_, err = logic.Like(likeReq(2, 300))
	assert.NoError(t, err)
}

// TestLike_SameUserDifferentContent 同用户点赞不同内容各自独立
func TestLike_SameUserDifferentContent(t *testing.T) {
	logic, mr := newTestLikeLogic(t)

	_, err := logic.Like(likeReq(5, 401))
	require.NoError(t, err)

	_, err = logic.Like(likeReq(5, 402))
	require.NoError(t, err)

	userLikeKey := "like:user:5"
	field401 := mr.HGet(userLikeKey, "401")
	field402 := mr.HGet(userLikeKey, "402")
	assert.Equal(t, "1", field401)
	assert.Equal(t, "1", field402)
}
