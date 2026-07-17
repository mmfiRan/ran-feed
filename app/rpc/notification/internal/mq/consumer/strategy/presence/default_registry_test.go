package presence

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/notification/internal/mq/consumer/strategy"
)

// 用 presence 包做为 strategy.NewDefaultRegistry 的四表注册验证宿主
// 放在此处避免 strategy 测试文件 import presence 造成 cycle
func TestDefaultRegistry_四张源表全注册(t *testing.T) {
	reg := strategy.NewDefaultRegistry()
	for _, table := range []string{"ran_feed_like", "ran_feed_favorite", "ran_feed_comment", "ran_feed_follow"} {
		_, ok := reg.Get(table)
		assert.True(t, ok, "策略 %s 未注册", table)
	}

	_, ok := reg.Get("ran_feed_user")
	assert.False(t, ok, "未监听表应返 false")
}
