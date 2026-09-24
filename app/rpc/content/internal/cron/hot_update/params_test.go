package hot_update

import (
	"fmt"
	"testing"

	"ran-feed/pkg/consts"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseParams_ShardConsistency 任务按 p.Shards 冻结并清理分片
// 缺省须取约定值 显式与写入方不一致必须报错 否则落在其余分片的脏内容永不被处理
func TestParseParams_ShardConsistency(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "空参数取约定值", raw: ""},
		{name: "显式零取约定值", raw: `{"shards":0}`},
		{name: "显式一致通过", raw: fmt.Sprintf(`{"shards":%d}`, consts.HotDirtyShards)},
		{name: "显式偏小报错", raw: fmt.Sprintf(`{"shards":%d}`, consts.HotDirtyShards-1), wantErr: true},
		{name: "显式偏大报错", raw: fmt.Sprintf(`{"shards":%d}`, consts.HotDirtyShards+1), wantErr: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			p, err := parseParams(tt.raw)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, consts.HotDirtyShards, p.Shards)
		})
	}
}

// TestParseParams_DefaultsApplied 其余缺省项按内置默认值补齐 窗口从半衰期推导
func TestParseParams_DefaultsApplied(t *testing.T) {
	p, err := parseParams("")
	require.NoError(t, err)
	assert.Equal(t, defaultTopN, p.TopN)
	assert.Equal(t, defaultMainN, p.MainN)
	assert.Equal(t, defaultFullLockTTL, p.LockTTL)
	assert.Equal(t, defaultHalfLifeHour, int(p.HalfLifeHours))
	assert.Equal(t, defaultBatchSize, p.BatchSize)
	assert.Equal(t, defaultPageSize, p.PageSize)
	assert.Equal(t, deriveWindowDays(defaultHalfLifeHour), p.WindowDays)
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
