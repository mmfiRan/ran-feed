package contentservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/svc"
)

const publishHotSeedDelta = 2.4

func shouldSeedHotIncrement(visibility content.Visibility) bool {
	return visibility == content.Visibility_PUBLIC
}

func writePublishHotSeed(ctx context.Context, svcCtx *svc.ServiceContext, contentID int64) error {
	if contentID <= 0 {
		return nil
	}
	shard := int(contentID % int64(rediskey.RedisFeedHotIncDefaultShards))
	incKey := rediskey.BuildHotFeedIncKey(shard)
	_, err := svcCtx.Redis.HincrbyFloatCtx(ctx, incKey, strconv.FormatInt(contentID, 10), publishHotSeedDelta)
	return err
}
