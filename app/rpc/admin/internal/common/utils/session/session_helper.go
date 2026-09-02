// Package session admin 登录态会话工具
package session

import (
	"context"

	"ran-feed/app/rpc/admin/internal/common/consts"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// RemoveByAdminID 管理员踢下线 读反向索引拿 token 批量删双向 key 失败返回 err
func RemoveByAdminID(ctx context.Context, r *redis.Redis, adminID int64) error {
	adminKey := consts.BuildAdminSessionUIDKey(adminID)
	token, err := r.GetCtx(ctx, adminKey)
	if err != nil {
		return err
	}
	if token == "" {
		return nil
	}
	return r.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		pipe.Del(ctx, consts.BuildAdminSessionKey(token))
		pipe.Del(ctx, adminKey)
		return nil
	})
}
