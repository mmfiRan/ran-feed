package feedprojector

import (
	"context"
	"strconv"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
)

// cleanup 删除或下架清理热榜主榜 加 作者发件箱 加 失效 L2 follower inbox
func (p *Projector) cleanup(ctx context.Context, contentID, authorID int64) error {
	contentIDStr := strconv.FormatInt(contentID, 10)
	if _, err := p.redis.ZremCtx(ctx, rediskey.RedisFeedHotGlobalKey, contentIDStr); err != nil {
		return err
	}
	if authorID > 0 {
		if _, err := p.redis.ZremCtx(ctx, rediskey.BuildUserPublishFeedKey(authorID), contentIDStr); err != nil {
			return err
		}
	}
	return p.contentCache.Invalidate(ctx, contentID)
}
