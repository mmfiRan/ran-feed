package rbac

import (
	"context"

	"ran-feed/app/admin/internal/common/consts"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// PermissionLoader 回源加载管理员权限点 code
type PermissionLoader func(ctx context.Context, adminID int64) ([]string, error)

// LoadPermissions cache-aside 取管理员权限点集合 命中读 Redis set 未命中回源写缓存
// 写入哨兵成员区分缓存未命中与真无权限 缓存故障只降级不阻断
func LoadPermissions(ctx context.Context, r *redis.Redis, adminID int64, ttlSeconds int, loader PermissionLoader) (map[string]struct{}, error) {
	key := consts.BuildAdminPermKey(adminID)

	if members, err := r.SmembersCtx(ctx, key); err == nil && len(members) > 0 {
		return toSet(members), nil
	}

	codes, err := loader(ctx, adminID)
	if err != nil {
		return nil, err
	}

	values := make([]any, 0, len(codes)+1)
	values = append(values, consts.RedisAdminPermLoadedSentinel)
	for _, c := range codes {
		values = append(values, c)
	}
	if _, aErr := r.SaddCtx(ctx, key, values...); aErr != nil {
		logx.WithContext(ctx).Errorf("写权限缓存失败 adminID=%d err=%v", adminID, aErr)
	} else {
		_ = r.ExpireCtx(ctx, key, ttlSeconds)
	}

	return toSet(codes), nil
}

// Invalidate 失效管理员权限缓存 角色/权限变更后调用
func Invalidate(ctx context.Context, r *redis.Redis, adminID int64) error {
	_, err := r.DelCtx(ctx, consts.BuildAdminPermKey(adminID))
	return err
}
