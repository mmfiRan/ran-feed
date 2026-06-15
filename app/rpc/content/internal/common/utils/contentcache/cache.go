// Package contentcache 提供 feed 内容详情的 Redis 二级缓存 旁路 cache-aside
package contentcache

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/config"
	"ran-feed/app/rpc/content/internal/do"
)

// Loader 缓存 miss 时的批量回源 只会传入未命中的 content_id
// 查不到的 id 不放进返回 map 由缓存层写负哨兵
type Loader func(missIDs []int64) (map[int64]*do.ContentDetailDO, error)

// BatchGet 批量查内容详情 cache-aside 自动去重并跳过 id<=0 返回 map 的 key 是 content_id
// loader 只会被传入缓存 miss 的 id Redis 任何错误均降级回源
func BatchGet(ctx context.Context, rds *redis.Redis, cfg config.ContentCacheConfig, contentIDs []int64, loader Loader) (map[int64]*do.ContentDetailDO, error) {
	result := make(map[int64]*do.ContentDetailDO, len(contentIDs))
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

	// 缓存关闭时直接回源
	if !cacheEnabled(cfg) {
		return loader(uniqIDs)
	}

	cacheKeys := make([]string, 0, len(uniqIDs))
	for _, id := range uniqIDs {
		cacheKeys = append(cacheKeys, rediskey.BuildContentDetailKey(id))
	}

	missIDs := uniqIDs
	cacheVals, err := rds.MgetCtx(ctx, cacheKeys...)
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
			d := &do.ContentDetailDO{}
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
	writeBackBatch(ctx, rds, cfg, missIDs, detailMap, logger)
	return result, nil
}

// Invalidate 删除指定内容的详情缓存 供内容编辑/删除/改可见性写路径调用
func Invalidate(ctx context.Context, rds *redis.Redis, contentIDs ...int64) error {
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
	_, err := rds.DelCtx(ctx, keys...)
	return err
}

func cacheEnabled(cfg config.ContentCacheConfig) bool {
	return cfg.TTLSeconds > 0
}

func writeBackBatch(ctx context.Context, rds *redis.Redis, cfg config.ContentCacheConfig, missIDs []int64, detailMap map[int64]*do.ContentDetailDO, logger logx.Logger) {
	type entry struct {
		key   string
		value string
		ttl   int
	}
	entries := make([]entry, 0, len(missIDs))
	for _, id := range missIDs {
		value, ttl, err := encodeForCache(cfg, detailMap[id])
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
	if err := rds.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		for _, e := range entries {
			pipe.Set(ctx, e.key, e.value, time.Duration(e.ttl)*time.Second)
		}
		return nil
	}); err != nil {
		logger.Errorf("批量回填内容详情缓存失败 err=%v", err)
	}
}

// encodeForCache 把详情序列化为 (value ttl) d==nil 时返回负哨兵与 negative TTL
func encodeForCache(cfg config.ContentCacheConfig, d *do.ContentDetailDO) (string, int, error) {
	if d == nil {
		return rediskey.RedisContentDetailMissingSentinel, negativeExpireWithJitter(cfg), nil
	}
	b, err := json.Marshal(d)
	if err != nil {
		return "", 0, err
	}
	return string(b), expireWithJitter(cfg), nil
}

func expireWithJitter(cfg config.ContentCacheConfig) int {
	base := int(cfg.TTLSeconds)
	if base <= 0 {
		return 0
	}
	jmax := int(cfg.JitterMaxSeconds)
	if jmax <= 0 {
		return base
	}
	return base + rand.Intn(jmax+1)
}

func negativeExpireWithJitter(cfg config.ContentCacheConfig) int {
	base := int(cfg.NegativeTTLSeconds)
	if base <= 0 {
		base = 60
	}
	jmax := int(cfg.NegativeJitterMaxSeconds)
	if jmax <= 0 {
		return base
	}
	return base + rand.Intn(jmax+1)
}
