package contentservicelogic

import (
	"context"
	"strconv"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/svc"
)

func shouldSeedHotIncrement(visibility content.Visibility) bool {
	return visibility == content.Visibility_PUBLIC
}

// writePublishHotSeed 发布即登记脏 把新内容放进热榜脏集合 让下一轮快更算分
// 0 互动新内容靠加法时间项进 TopN 冷启动自带解药 见 HOT_FEED_DESIGN 第2节 无需额外 seed 分值
func writePublishHotSeed(ctx context.Context, svcCtx *svc.ServiceContext, contentID int64) error {
	if contentID <= 0 {
		return nil
	}
	shard := int(contentID % int64(rediskey.RedisFeedHotIncDefaultShards))
	dirtyKey := rediskey.BuildHotFeedDirtyKey(shard)
	_, err := svcCtx.Redis.SaddCtx(ctx, dirtyKey, strconv.FormatInt(contentID, 10))
	return err
}
