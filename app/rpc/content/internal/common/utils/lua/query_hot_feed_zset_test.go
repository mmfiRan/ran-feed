package lua

import (
	"context"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// TestQueryHotFeedZSet_TiedScoreCrossDigitPagination
// fix-012 回归：当多个 content_id 共享同一分数且位数不同时，
// 旧版本用 tonumber(member) >= cursorId 做数字比较会与 Redis 同分组内字典序排序错位，
// 导致第二页丢失 lex 排序在 cursor 之后的成员（例如 "100"）。
// 新版本应严格按字典序过滤，全部成员能在多页之间完整返回且无重复。
func TestQueryHotFeedZSet_TiedScoreCrossDigitPagination(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	globalKey := "feed:hot:global:zset"

	// 同分数 1.0，三个位数不同的 content_id
	for _, id := range []string{"9", "20", "100"} {
		_, err := r.ZaddCtx(context.Background(), globalKey, 1, id)
		require.NoError(t, err)
	}

	const pageSize = 2

	// 使用 preferredKey 直通 globalKey，避开脚本中 latest 分支无关边界。
	keys := []string{globalKey, "feed:hot:global:latest", "feed:hot:global:snap", globalKey}

	// 第一页
	res1, err := r.EvalCtx(
		context.Background(),
		QueryHotFeedZSetScript,
		keys,
		"",
		strconv.Itoa(pageSize),
		"",
	)
	require.NoError(t, err)
	page1 := extractIDs(t, res1)
	// ZREVRANGEBYSCORE 同分组内字典序降序：'9' > '20' > '100'
	assert.Equal(t, []string{"9", "20"}, page1.ids, "first page IDs")
	assert.Equal(t, int64(1), page1.hasMore, "first page hasMore")
	assert.Equal(t, "20", page1.nextCursor, "first page nextCursor")

	// 第二页
	res2, err := r.EvalCtx(
		context.Background(),
		QueryHotFeedZSetScript,
		keys,
		page1.nextCursor,
		strconv.Itoa(pageSize),
		"",
	)
	require.NoError(t, err)
	page2 := extractIDs(t, res2)
	// 旧版本 bug 下这里会返回 ["9"]（页 1 重复），且 "100" 永远丢失。
	// 修复后应返回剩余的 "100" 一项，hasMore=0。
	assert.Equal(t, []string{"100"}, page2.ids, "second page should contain the lex-smaller tied member")
	assert.Equal(t, int64(0), page2.hasMore, "second page hasMore")
}

// TestQueryHotFeedZSet_LatestMissedFallsThroughToGlobal
// fix-015 回归：latestKey 不存在时 redis.call('GET') 返回 Lua false（不是 nil/""），
// 旧代码 `latestId ~= nil and latestId ~= ""` 让 false 通过判断后在下一行 concat 崩溃。
// 修复后应安全 fall through 到 globalKey，并返回 globalKey 上的内容。
func TestQueryHotFeedZSet_LatestMissedFallsThroughToGlobal(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	globalKey := "feed:hot:global:zset"
	_, err = r.ZaddCtx(context.Background(), globalKey, 5, "100")
	require.NoError(t, err)
	_, err = r.ZaddCtx(context.Background(), globalKey, 3, "200")
	require.NoError(t, err)

	// preferredKey 为空 + latestKey 不存在 → 走进 latest 分支但 GET 返回 false。
	// 旧版本会在 line 33 concat 崩溃，新版本应识别 false 后 fall through 到 globalKey。
	res, err := r.EvalCtx(
		context.Background(),
		QueryHotFeedZSetScript,
		[]string{"", "feed:hot:global:latest:nonexistent", "feed:hot:global:snap", globalKey},
		"",
		"10",
		"",
	)
	require.NoError(t, err, "latest 分支 false 不应再崩溃")
	page := extractIDs(t, res)
	assert.Equal(t, int64(1), page.exists)
	assert.Equal(t, []string{"100", "200"}, page.ids)
}

type hotFeedPage struct {
	exists     int64
	hasMore    int64
	nextCursor string
	ids        []string
}

func extractIDs(t *testing.T, raw interface{}) hotFeedPage {
	t.Helper()
	arr, ok := raw.([]interface{})
	require.True(t, ok, "expect script to return an array, got %T", raw)
	require.GreaterOrEqual(t, len(arr), 4)

	page := hotFeedPage{}
	page.exists, _ = arr[0].(int64)
	page.hasMore, _ = arr[1].(int64)
	if s, ok := arr[2].(string); ok {
		page.nextCursor = s
	}
	for i := 4; i < len(arr); i++ {
		if s, ok := arr[i].(string); ok {
			page.ids = append(page.ids, s)
		}
	}
	return page
}