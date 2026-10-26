package bigv

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/stores/redis"

	sharedkey "ran-feed/pkg/rediskey"
)

func newTestSet(t *testing.T) (*Set, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	r := redis.MustNewRedis(redis.RedisConf{Host: mr.Addr(), Type: redis.NodeType})
	return New(r), mr
}

func TestFilter(t *testing.T) {
	tests := []struct {
		name       string
		members    []string
		candidates []int64
		want       []int64
	}{
		{name: "命中保持候选顺序", members: []string{"2", "4"}, candidates: []int64{4, 1, 2, 3}, want: []int64{4, 2}},
		{name: "全不命中返回空", members: nil, candidates: []int64{1, 2}, want: []int64{}},
		{name: "候选为空返回 nil", candidates: nil, want: nil},
		{name: "非法 id 跳过", members: []string{"1"}, candidates: []int64{0, -1, 1}, want: []int64{1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, mr := newTestSet(t)
			if len(tt.members) > 0 {
				_, err := mr.SAdd(sharedkey.FeedBigVGlobal, tt.members...)
				require.NoError(t, err)
			}

			got, err := s.Filter(context.Background(), tt.candidates)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
