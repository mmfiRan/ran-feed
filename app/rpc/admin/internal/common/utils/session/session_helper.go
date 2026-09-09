// Package session admin 登录态会话工具
package session

import (
	"context"
	"strconv"
	"time"

	"ran-feed/app/rpc/admin/internal/common/consts"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

// SaveSession 保存登陆会话
func SaveSession(ctx context.Context, r *redis.Redis, adminID int64, token string, ttlSeconds int) error {
	tokenKey := consts.BuildAdminSessionKey(token)
	adminKey := consts.BuildAdminSessionUIDKey(adminID)

	oldToken, err := r.GetCtx(ctx, adminKey)
	if err != nil {
		return err
	}

	return r.PipelinedCtx(ctx, func(pipe redis.Pipeliner) error {
		if oldToken != "" {
			pipe.Del(ctx, consts.BuildAdminSessionKey(oldToken))
		}
		pipe.SetEx(ctx, tokenKey, strconv.FormatInt(adminID, 10), time.Duration(ttlSeconds)*time.Second)
		pipe.SetEx(ctx, adminKey, token, time.Duration(ttlSeconds)*time.Second)
		return nil
	})
}

// RemoveByAdminID 管理员踢下线
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
