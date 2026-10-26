// Package bigv 负责全局大V集合的判定
package bigv

import (
	"context"
	"fmt"
	"strconv"

	sharedkey "ran-feed/pkg/rediskey"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// Set 全局大V集合的只读入口
type Set struct {
	redis *redis.Redis
}

func New(redisClient *redis.Redis) *Set {
	return &Set{
		redis: redisClient,
	}
}

// IsBigV 判断作者是否是大V
func (s *Set) IsBigV(ctx context.Context, authorID int64) (bool, error) {
	if authorID <= 0 {
		return false, nil
	}
	return s.redis.SismemberCtx(ctx, sharedkey.FeedBigVGlobal, strconv.FormatInt(authorID, 10))
}

// Filter 批量筛出候选里的大V,用 SMISMEMBER 一次往返拿全部标记,返回保持候选顺序
func (s *Set) Filter(ctx context.Context, candidates []int64) ([]int64, error) {
	ids := make([]int64, 0, len(candidates))
	args := make([]any, 0, len(candidates)+2)
	args = append(args, "SMISMEMBER", sharedkey.FeedBigVGlobal)
	for _, id := range candidates {
		if id <= 0 {
			continue
		}
		ids = append(ids, id)
		args = append(args, strconv.FormatInt(id, 10))
	}
	if len(ids) == 0 {
		return nil, nil
	}

	res, err := s.redis.DoCtx(ctx, args...)
	if err != nil {
		return nil, err
	}
	flags, ok := res.([]any)
	if !ok || len(flags) != len(ids) {
		return nil, fmt.Errorf("SMISMEMBER 返回非法 want=%d got=%d", len(ids), len(flags))
	}

	hit := make([]int64, 0, len(ids))
	for i, v := range flags {
		if n, ok := v.(int64); ok && n == 1 {
			hit = append(hit, ids[i])
		}
	}
	return hit, nil
}
