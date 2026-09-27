package contentcache

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
)

const (
	ttlSeconds         = 600
	negativeTTLSeconds = 60
)

// withJitter TTL 加 0 到 ttl 的随机抖动 过期落在 ttl 到两倍 ttl 之间 抗雪崩
// 上界由 ttl 推出 不单独配常量 免得改 ttl 时漏改上界
func withJitter(ttlSeconds int) int {
	return ttlSeconds + rand.Intn(ttlSeconds+1)
}

// writeBackBatch 回源结果批量写回 查不到的写负哨兵配一个短 TTL 写失败只记日志不影响调用方
func (c *Cache) writeBackBatch(ctx context.Context, missIDs []int64, detailMap map[int64]*Detail, logger logx.Logger) {
	type entry struct {
		key   string
		value string
		ttl   int
	}
	entries := make([]entry, 0, len(missIDs))
	for _, id := range missIDs {
		value, ttl, err := encodeForCache(detailMap[id])
		if err != nil {
			logger.Errorf("序列化内容详情缓存失败 id=%d err=%v", id, err)
			continue
		}
		entries = append(entries, entry{
			key:   rediskey.BuildContentDetailKey(id),
			value: value,
			ttl:   ttl,
		})
	}
	if len(entries) == 0 {
		return
	}
	if err := c.redis.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		for _, e := range entries {
			pipe.Set(ctx, e.key, e.value, time.Duration(e.ttl)*time.Second)
		}
		return nil
	}); err != nil {
		logger.Errorf("批量回填内容详情缓存失败 err=%v", err)
	}
}

// encodeForCache 序列化缓存值并给出 TTL d 为 nil 说明这条内容查不到 写负哨兵配一个短 TTL
func encodeForCache(d *Detail) (string, int, error) {
	if d == nil {
		return rediskey.RedisContentDetailMissingSentinel, withJitter(negativeTTLSeconds), nil
	}
	b, err := json.Marshal(d)
	if err != nil {
		return "", 0, err
	}
	return string(b), withJitter(ttlSeconds), nil
}
