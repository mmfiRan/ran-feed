package consumer

import (
	"context"
	"strconv"
	"testing"

	rediskey "ran-feed/app/rpc/count/internal/common/consts/redis"
	"ran-feed/app/rpc/count/internal/svc"
	"ran-feed/pkg/consts"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

func newTestCountConsumer(t *testing.T) (*miniredis.Miniredis, *redis.Redis, *CanalCountConsumer) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	return mr, r, &CanalCountConsumer{
		ctx:        context.Background(),
		svcContext: &svc.ServiceContext{Redis: r},
		Logger:     logx.WithContext(context.Background()),
	}
}

// TestMarkHotDirty_ShardedByMod 内容按 id 取模落到对应分片 一次 pipeline 提交
func TestMarkHotDirty_ShardedByMod(t *testing.T) {
	_, r, c := newTestCountConsumer(t)
	cs := newChangeSet()
	cs.contents[1001] = struct{}{}
	cs.contents[1002] = struct{}{}
	cs.contents[1001+consts.HotDirtyShards] = struct{}{}

	require.NoError(t, c.markHotDirty(context.Background(), cs))

	for id := range cs.contents {
		shard := int(id % int64(consts.HotDirtyShards))
		member, err := r.SismemberCtx(context.Background(), rediskey.BuildHotFeedDirtyKey(shard), strconv.FormatInt(id, 10))
		require.NoError(t, err)
		assert.True(t, member, "contentID=%d 应落在分片 %d", id, shard)
	}
}

// TestMarkHotDirty_EmptyNoWrite 空变更集不应写任何 key
func TestMarkHotDirty_EmptyNoWrite(t *testing.T) {
	mr, _, c := newTestCountConsumer(t)
	require.NoError(t, c.markHotDirty(context.Background(), newChangeSet()))
	assert.Empty(t, mr.Keys())
}

// TestMarkHotDirty_SkipsNonPositive 非正 id 不登记
func TestMarkHotDirty_SkipsNonPositive(t *testing.T) {
	mr, _, c := newTestCountConsumer(t)
	cs := newChangeSet()
	cs.contents[0] = struct{}{}
	cs.contents[-5] = struct{}{}

	require.NoError(t, c.markHotDirty(context.Background(), cs))
	assert.Empty(t, mr.Keys())
}
