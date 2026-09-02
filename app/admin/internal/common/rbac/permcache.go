package rbac

import (
	"context"
	"time"

	"ran-feed/app/admin/internal/common/consts"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// PermissionLoader 回源加载管理员权限点 code
type PermissionLoader func(ctx context.Context, adminID int64) ([]string, error)

// LoadPermissions 取管理员权限点集合
func LoadPermissions(ctx context.Context, r *redis.Redis, adminID int64, ttlSeconds int, loader PermissionLoader) (map[string]struct{}, error) {
	key := consts.BuildAdminPermKey(adminID)

	if members, err := r.SmembersCtx(ctx, key); err == nil && len(members) > 0 {
		err = r.ExpireCtx(ctx, key, ttlSeconds)
		if err != nil {
			logx.WithContext(ctx).Errorf("刷新权限缓存失败 adminID=%d err=%v", adminID, err)
		}
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

	if err = r.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		pipe.SAdd(ctx, key, values...)
		pipe.Expire(ctx, key, time.Duration(ttlSeconds)*time.Second)
		return nil
	}); err != nil {
		logx.WithContext(ctx).Errorf("写权限缓存失败 adminID=%d err=%v", adminID, err)
	}

	return toSet(codes), nil
}
