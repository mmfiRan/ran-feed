// Package usercache 提供用户信息的 Redis 旁路缓存。
//
// 三个公开查询接口 GetUser / BatchGetUser / GetUserProfile 共用同一 DO
// 来源（repo.GetByID / BatchGetByIDs），统一收敛到此层避免重复实现。
//
// 策略：
//   - 命中正值：JSON 反序列化为 *do.UserDO 返回
//   - 命中负值哨兵（"-"）：返回 (nil, nil)，不再回源 DB
//   - miss：回源 DB，成功写正值，DB 也 miss 时写哨兵
//
// TTL 正负值均叠加 [0, JitterMax] 抖动抗雪崩，与 count 服务保持一致风格。
package usercache

import (
	"context"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	rediskey "ran-feed/app/rpc/user/internal/common/consts/redis"
	"ran-feed/app/rpc/user/internal/config"
	"ran-feed/app/rpc/user/internal/do"
	"ran-feed/app/rpc/user/internal/repositories"
)

// userCacheDO 缓存中存放的字段子集，去掉了 PasswordHash / PasswordSalt /
// Email 等敏感字段（这些字段调用方接口从不返回）。
type userCacheDO struct {
	ID        int64      `json:"id"`
	Username  string     `json:"username"`
	Nickname  string     `json:"nickname"`
	Avatar    string     `json:"avatar"`
	Bio       string     `json:"bio"`
	Mobile    string     `json:"mobile"`
	Gender    int32      `json:"gender"`
	Birthday  *time.Time `json:"birthday,omitempty"`
	Status    int32      `json:"status"`
	CreatedBy int64      `json:"created_by"`
	UpdatedBy int64      `json:"updated_by"`
}

func toCacheDO(u *do.UserDO) *userCacheDO {
	return &userCacheDO{
		ID:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Bio:       u.Bio,
		Mobile:    u.Mobile,
		Gender:    u.Gender,
		Birthday:  u.Birthday,
		Status:    u.Status,
		CreatedBy: u.CreatedBy,
		UpdatedBy: u.UpdatedBy,
	}
}

func (c *userCacheDO) toUserDO() *do.UserDO {
	return &do.UserDO{
		ID:        c.ID,
		Username:  c.Username,
		Nickname:  c.Nickname,
		Avatar:    c.Avatar,
		Bio:       c.Bio,
		Mobile:    c.Mobile,
		Gender:    c.Gender,
		Birthday:  c.Birthday,
		Status:    c.Status,
		CreatedBy: c.CreatedBy,
		UpdatedBy: c.UpdatedBy,
	}
}

// Get 单查用户信息，cache-aside。
// userID <= 0 直接返回 (nil, nil)。
// 缓存层任何 Redis 错误均降级为直接回源，错误只记日志不冒泡。
func Get(
	ctx context.Context,
	rds *redis.Redis,
	repo repositories.UserRepository,
	cfg config.UserCacheConfig,
	userID int64,
) (*do.UserDO, error) {
	if userID <= 0 {
		return nil, nil
	}
	logger := logx.WithContext(ctx)

	// 缓存关闭时直接走 DB
	if !cacheEnabled(cfg) {
		return repo.GetByID(userID)
	}

	cacheKey := rediskey.BuildUserInfoKey(userID)
	if u, hit := loadFromCache(ctx, rds, cacheKey, logger); hit {
		return u, nil
	}

	u, err := repo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	writeBack(ctx, rds, cfg, cacheKey, u, logger)
	return u, nil
}

// BatchGet 批量查询用户信息，cache-aside。
// 自动去重并跳过 userID <= 0 的元素。返回 map 的 key 是用户 ID。
// Redis MGET 失败时整体降级回 DB（保留单次批查特性）。
func BatchGet(
	ctx context.Context,
	rds *redis.Redis,
	repo repositories.UserRepository,
	cfg config.UserCacheConfig,
	userIDs []int64,
) (map[int64]*do.UserDO, error) {
	result := make(map[int64]*do.UserDO, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}

	uniqIDs := make([]int64, 0, len(userIDs))
	seen := make(map[int64]struct{}, len(userIDs))
	for _, id := range userIDs {
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

	if !cacheEnabled(cfg) {
		dbMap, err := repo.BatchGetByIDs(uniqIDs)
		if err != nil {
			return nil, err
		}
		return dbMap, nil
	}

	cacheKeys := make([]string, 0, len(uniqIDs))
	for _, id := range uniqIDs {
		cacheKeys = append(cacheKeys, rediskey.BuildUserInfoKey(id))
	}

	missIDs := uniqIDs
	cacheVals, err := rds.MgetCtx(ctx, cacheKeys...)
	if err != nil {
		logger.Errorf("批量查询用户信息缓存失败: %v", err)
	} else if len(cacheVals) == len(uniqIDs) {
		missIDs = missIDs[:0]
		for i, id := range uniqIDs {
			raw := cacheVals[i]
			if raw == "" {
				missIDs = append(missIDs, id)
				continue
			}
			if raw == rediskey.RedisUserInfoMissingSentinel {
				continue
			}
			c := &userCacheDO{}
			if perr := json.Unmarshal([]byte(raw), c); perr != nil {
				logger.Errorf("解析用户缓存失败: id=%d, err=%v", id, perr)
				missIDs = append(missIDs, id)
				continue
			}
			result[id] = c.toUserDO()
		}
	}

	if len(missIDs) == 0 {
		return result, nil
	}

	dbMap, err := repo.BatchGetByIDs(missIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range missIDs {
		if u, ok := dbMap[id]; ok && u != nil {
			result[id] = u
		}
	}
	writeBackBatch(ctx, rds, cfg, missIDs, dbMap, logger)
	return result, nil
}

// Invalidate 删除指定用户的缓存，供未来 update 路径调用。
func Invalidate(ctx context.Context, rds *redis.Redis, userID int64) error {
	if userID <= 0 {
		return nil
	}
	_, err := rds.DelCtx(ctx, rediskey.BuildUserInfoKey(userID))
	return err
}

func cacheEnabled(cfg config.UserCacheConfig) bool {
	return cfg.TTLSeconds > 0
}

func loadFromCache(ctx context.Context, rds *redis.Redis, cacheKey string, logger logx.Logger) (*do.UserDO, bool) {
	raw, err := rds.GetCtx(ctx, cacheKey)
	if err != nil {
		logger.Errorf("查询用户缓存失败: key=%s, err=%v", cacheKey, err)
		return nil, false
	}
	if raw == "" {
		return nil, false
	}
	if raw == rediskey.RedisUserInfoMissingSentinel {
		return nil, true
	}
	c := &userCacheDO{}
	if perr := json.Unmarshal([]byte(raw), c); perr != nil {
		logger.Errorf("解析用户缓存失败: key=%s, err=%v", cacheKey, perr)
		return nil, false
	}
	return c.toUserDO(), true
}

func writeBack(
	ctx context.Context,
	rds *redis.Redis,
	cfg config.UserCacheConfig,
	cacheKey string,
	u *do.UserDO,
	logger logx.Logger,
) {
	value, ttl, err := encodeForCache(cfg, u)
	if err != nil {
		logger.Errorf("序列化用户缓存失败: key=%s, err=%v", cacheKey, err)
		return
	}
	if err := rds.SetexCtx(ctx, cacheKey, value, ttl); err != nil {
		logger.Errorf("回填用户缓存失败: key=%s, err=%v", cacheKey, err)
	}
}

func writeBackBatch(
	ctx context.Context,
	rds *redis.Redis,
	cfg config.UserCacheConfig,
	missIDs []int64,
	dbMap map[int64]*do.UserDO,
	logger logx.Logger,
) {
	type entry struct {
		key   string
		value string
		ttl   int
	}
	entries := make([]entry, 0, len(missIDs))
	for _, id := range missIDs {
		value, ttl, err := encodeForCache(cfg, dbMap[id])
		if err != nil {
			logger.Errorf("序列化用户缓存失败: id=%d, err=%v", id, err)
			continue
		}
		entries = append(entries, entry{
			key:   rediskey.BuildUserInfoKey(id),
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
		logger.Errorf("批量回填用户缓存失败: %v", err)
	}
}

// encodeForCache 把 DO 序列化为 (value, ttl)；u==nil 时返回哨兵和 negative TTL。
func encodeForCache(cfg config.UserCacheConfig, u *do.UserDO) (string, int, error) {
	if u == nil {
		return rediskey.RedisUserInfoMissingSentinel, negativeExpireWithJitter(cfg), nil
	}
	b, err := json.Marshal(toCacheDO(u))
	if err != nil {
		return "", 0, err
	}
	return string(b), expireWithJitter(cfg), nil
}

func expireWithJitter(cfg config.UserCacheConfig) int {
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

func negativeExpireWithJitter(cfg config.UserCacheConfig) int {
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