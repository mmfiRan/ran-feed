package hotfeed

import (
	"testing"

	"ran-feed/pkg/consts"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNormalize_DefaultsApplied 零值参数按内置默认补齐 窗口从半衰期推导
func TestNormalize_DefaultsApplied(t *testing.T) {
	opts, err := Options{}.Normalize()
	require.NoError(t, err)
	assert.Equal(t, consts.HotDirtyShards, opts.Shards)
	assert.Equal(t, defaultTopN, opts.TopN)
	assert.Equal(t, defaultMainN, opts.MainN)
	assert.Equal(t, defaultHalfLifeHour, int(opts.HalfLifeHours))
	assert.Equal(t, defaultBatchSize, opts.BatchSize)
	assert.Equal(t, defaultPageSize, opts.PageSize)
	assert.Equal(t, deriveWindowDays(defaultHalfLifeHour), opts.WindowDays)
}

// TestNormalize_Idempotent 归一化可重复调用 二次结果不再变化
func TestNormalize_Idempotent(t *testing.T) {
	once, err := Options{}.Normalize()
	require.NoError(t, err)
	twice, err := once.Normalize()
	require.NoError(t, err)
	assert.Equal(t, once, twice)
}

// TestNormalize_MainNCoversTopN 主榜候选池必须不小于对外快照 否则没有候补垫 退化成主榜等于快照
func TestNormalize_MainNCoversTopN(t *testing.T) {
	opts, err := Options{MainN: 10, TopN: 100}.Normalize()
	require.NoError(t, err)
	assert.Equal(t, 100, opts.MainN)
}

// TestNormalize_ShardMismatchRejected 分片数与写入方不一致必须报错 否则其余分片的脏内容永不被处理
func TestNormalize_ShardMismatchRejected(t *testing.T) {
	_, err := Options{Shards: consts.HotDirtyShards + 1}.Normalize()
	require.Error(t, err)

	_, err = Options{Shards: consts.HotDirtyShards - 1}.Normalize()
	require.Error(t, err)
}

// TestDeriveWindowDays 窗口从半衰期推导 非正退化兜底 超上限封顶防全表扫
func TestDeriveWindowDays(t *testing.T) {
	cases := []struct {
		halfLifeHours float64
		want          int
	}{
		{0, defaultWindowDays},
		{-5, defaultWindowDays},
		{24, 17},
		{48, 34},
		{72, 50},
		{168, maxWindowDays},
		{720, maxWindowDays},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, deriveWindowDays(c.halfLifeHours), "deriveWindowDays(%.0f)", c.halfLifeHours)
	}
}
