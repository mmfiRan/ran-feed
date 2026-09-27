package hotupdate

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

// TestParseParams_LockTTLDefault 锁 TTL 属任务级 缺省按任务内置值补齐
func TestParseParams_LockTTLDefault(t *testing.T) {
	p, err := parseParams("")
	require.NoError(t, err)
	assert.Equal(t, defaultFullLockTTL, p.LockTTL)
}

// TestParseParams_OptionsPopulated 榜单参数走内嵌 Options 展开 缺省由组件归一化补齐
func TestParseParams_OptionsPopulated(t *testing.T) {
	p, err := parseParams(`{"mainN":700,"topN":300,"halfLifeHours":48}`)
	require.NoError(t, err)
	assert.Equal(t, 700, p.MainN)
	assert.Equal(t, 300, p.TopN)
	assert.Equal(t, 48.0, p.HalfLifeHours)
	assert.Equal(t, defaultFullLockTTL, p.LockTTL)
	assert.NotZero(t, p.BatchSize)
	assert.NotZero(t, p.PageSize)
}
