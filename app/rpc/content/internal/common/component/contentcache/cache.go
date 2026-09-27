// Package contentcache 提供 feed 内容详情的 Redis 二级缓存
package contentcache

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
)

// Loader 缓存没命中时的批量回源 只会收到没命中的 id
type Loader func(missIDs []int64) (map[int64]*Detail, error)

// Cache 内容详情的二级缓存 命中直返 未命中回源并写回 查不到的写负哨兵
type Cache struct {
	redis *redis.Redis
}

func New(redisClient *redis.Redis) *Cache {
	return &Cache{
		redis: redisClient,
	}
}

// BatchGet 按 id 批量取详情 loader 只收到没命中的 id Redis 出任何错都降级回源 不阻断调用方
func (c *Cache) BatchGet(ctx context.Context, contentIDs []int64, loader Loader) (map[int64]*Detail, error) {
	result := make(map[int64]*Detail, len(contentIDs))
	if len(contentIDs) == 0 {
		return result, nil
	}

	uniqIDs := make([]int64, 0, len(contentIDs))
	seen := make(map[int64]struct{}, len(contentIDs))
	for _, id := range contentIDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqIDs = append(uniqIDs, id)
	}
	if len(uniqIDs) == 0 {
		return result, nil
	}

	logger := logx.WithContext(ctx)

	cacheKeys := make([]string, 0, len(uniqIDs))
	for _, id := range uniqIDs {
		cacheKeys = append(cacheKeys, rediskey.BuildContentDetailKey(id))
	}

	missIDs := uniqIDs
	cacheVals, err := c.redis.MgetCtx(ctx, cacheKeys...)
	if err != nil {
		logger.Errorf("批量查询内容详情缓存失败 err=%v", err)
	} else if len(cacheVals) == len(uniqIDs) {
		missIDs = missIDs[:0]
		for i, id := range uniqIDs {
			raw := cacheVals[i]
			if raw == "" {
				missIDs = append(missIDs, id)
				continue
			}
			if raw == rediskey.RedisContentDetailMissingSentinel {
				continue
			}
			d := &Detail{}
			if perr := json.Unmarshal([]byte(raw), d); perr != nil {
				logger.Errorf("解析内容详情缓存失败 id=%d err=%v", id, perr)
				missIDs = append(missIDs, id)
				continue
			}
			result[id] = d
		}
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	detailMap, err := loader(missIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range missIDs {
		if d, ok := detailMap[id]; ok && d != nil {
			result[id] = d
		}
	}
	c.writeBackBatch(ctx, missIDs, detailMap, logger)
	return result, nil
}

// Invalidate 删除指定内容的详情缓存
func (c *Cache) Invalidate(ctx context.Context, contentIDs ...int64) error {
	keys := make([]string, 0, len(contentIDs))
	for _, id := range contentIDs {
		if id <= 0 {
			continue
		}
		keys = append(keys, rediskey.BuildContentDetailKey(id))
	}
	if len(keys) == 0 {
		return nil
	}
	_, err := c.redis.DelCtx(ctx, keys...)
	return err
}
