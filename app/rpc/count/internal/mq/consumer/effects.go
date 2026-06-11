package consumer

import (
	"context"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/core/threading"

	rediskey "ran-feed/app/rpc/count/internal/common/consts/redis"
)

const (
	userProfileCacheInvalidateDelay   = 200 * time.Millisecond
	userProfileCacheInvalidateTimeout = 5 * time.Second
)

// dispatch 事务提交后批量派发副作用 失效计数与用户主页缓存并登记热榜脏 全部脱离请求 ctx
func (c *CanalCountConsumer) dispatch(ctx context.Context, cs *changeSet) error {
	if cs.empty() {
		return nil
	}
	c.invalidateCountCaches(cs)
	c.invalidateUserProfileCaches(cs)
	return c.markHotDirty(ctx, cs)
}

// invalidateCountCaches 旁路缓存 写成功后删计数缓存 即时加延迟双删由 operator 内部完成
func (c *CanalCountConsumer) invalidateCountCaches(cs *changeSet) {
	for key := range cs.counts {
		c.deltaOperator.InvalidateCountCache(key.bizType, key.targetType, key.targetID)
	}
}

// invalidateUserProfileCaches 失效受影响用户主页聚合计数缓存
func (c *CanalCountConsumer) invalidateUserProfileCaches(cs *changeSet) {
	for userID := range cs.users {
		if userID <= 0 {
			continue
		}
		cacheKey := rediskey.GetRedisPrefixKey(rediskey.RedisUserProfileCountsPrefix, strconv.FormatInt(userID, 10))
		c.delayedDoubleDelete(cacheKey)
	}
}

// delayedDoubleDelete 即时删一次再延迟删一次 堵住删后到写可见前的回填窗口
// 延迟删用 bg ctx 加超时 避免 consumer 关停时请求 ctx 取消让残留清理失败
func (c *CanalCountConsumer) delayedDoubleDelete(cacheKey string) {
	if _, err := c.svcContext.Redis.DelCtx(c.ctx, cacheKey); err != nil {
		c.Errorf("删除用户主页计数缓存失败: key=%s, err=%v", cacheKey, err)
	}
	threading.GoSafe(func() {
		time.Sleep(userProfileCacheInvalidateDelay)
		ctx, cancel := context.WithTimeout(context.Background(), userProfileCacheInvalidateTimeout)
		defer cancel()
		if _, err := c.svcContext.Redis.DelCtx(ctx, cacheKey); err != nil {
			c.Errorf("延迟删除用户主页计数缓存失败: key=%s, err=%v", cacheKey, err)
		}
	})
}

// markHotDirty 把发生互动的内容登记进热榜脏集合 按取模分片每片一次 SADD 只记谁脏了不算分
func (c *CanalCountConsumer) markHotDirty(ctx context.Context, cs *changeSet) error {
	if len(cs.contents) == 0 {
		return nil
	}
	byShard := make(map[int][]any, rediskey.RedisFeedHotIncDefaultShards)
	for contentID := range cs.contents {
		if contentID <= 0 {
			continue
		}
		shard := hotDirtyShard(contentID)
		byShard[shard] = append(byShard[shard], strconv.FormatInt(contentID, 10))
	}
	for shard, members := range byShard {
		dirtyKey := rediskey.BuildHotFeedDirtyKey(shard)
		if _, err := c.svcContext.Redis.SaddCtx(ctx, dirtyKey, members...); err != nil {
			return err
		}
	}
	return nil
}

func hotDirtyShard(contentID int64) int {
	if contentID <= 0 {
		return 0
	}
	return int(contentID % int64(rediskey.RedisFeedHotIncDefaultShards))
}