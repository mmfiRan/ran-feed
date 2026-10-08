// Package bigv 负责全局大 V 集合的判定
// 集合由计数域按 pkg/consts.BigVFollowerThreshold 维护 本域只读
package bigv

import (
	"context"
	"strconv"

	sharedkey "ran-feed/pkg/rediskey"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// Set 全局大 V 集合的只读入口
type Set struct {
	redis *redis.Redis
}

func New(redisClient *redis.Redis) *Set {
	return &Set{
		redis: redisClient,
	}
}

// IsBigV 单个作者是否大 V 查询失败由调用方决定保守取向
func (s *Set) IsBigV(ctx context.Context, authorID int64) (bool, error) {
	if authorID <= 0 {
		return false, nil
	}
	return s.redis.SismemberCtx(ctx, sharedkey.FeedBigVGlobal, strconv.FormatInt(authorID, 10))
}

// Filter 批量筛出候选里的大 V 单次 Lua 一个 RTT 不整体 SMEMBERS
// 成本随候选数增长 支撑棘轮语义下集合单调增长
func (s *Set) Filter(ctx context.Context, candidates []int64) ([]int64, error) {
	args := make([]any, 0, len(candidates))
	for _, id := range candidates {
		if id > 0 {
			args = append(args, strconv.FormatInt(id, 10))
		}
	}
	if len(args) == 0 {
		return nil, nil
	}

	res, err := s.redis.EvalCtx(ctx, filterMembersScript, []string{sharedkey.FeedBigVGlobal}, args...)
	if err != nil {
		return nil, err
	}
	arr, ok := res.([]any)
	if !ok {
		return nil, nil
	}

	hit := make([]int64, 0, len(arr))
	for _, v := range arr {
		id, ok := replyInt64(v)
		if !ok || id <= 0 {
			continue
		}
		hit = append(hit, id)
	}
	return hit, nil
}

// replyInt64 解本包脚本返回的 id Lua 的数字经 RESP 回来可能是整数也可能是字符串
func replyInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	case []byte:
		n, err := strconv.ParseInt(string(t), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}
