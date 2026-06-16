package contentservicelogic

import (
	"context"
	"math"
	"strconv"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	"ran-feed/app/rpc/content/internal/repositories"
	"ran-feed/app/rpc/content/internal/svc"
	"ran-feed/pkg/errorx"
)

// followeeContent followee 窗口内一条内容 backfill 与 purge 共用
type followeeContent struct {
	id          int64
	publishedAt int64
}

// isBigVAuthor 命中全局大 V 集合即大 V 读失败保守按非大 V 处理 backfill/purge/fanout 共用
func isBigVAuthor(ctx context.Context, svcCtx *svc.ServiceContext, authorID int64) (bool, error) {
	return svcCtx.Redis.SismemberCtx(ctx, rediskey.RedisFeedBigVGlobalKey, strconv.FormatInt(authorID, 10))
}

// loadFolloweeWindowContent 取 followee 窗口内内容 publish zset 优先 冷则回源 DB 取 PUBLIC
// backfill 灌入与 purge 清理共用同一取数口径 避免分叉
func loadFolloweeWindowContent(ctx context.Context, svcCtx *svc.ServiceContext, contentRepo repositories.ContentRepository, followeeID, cutoffMillis int64, limit int) ([]followeeContent, error) {
	publishKey := rediskey.BuildUserPublishFeedKey(followeeID)
	exists, err := svcCtx.Redis.ExistsCtx(ctx, publishKey)
	if err != nil {
		return nil, errorx.Wrap(ctx, err, errorx.NewMsg("查询关注者发布列表失败"))
	}
	if exists {
		pairs, perr := svcCtx.Redis.ZrevrangebyscoreWithScoresByFloatAndLimitCtx(ctx, publishKey, float64(cutoffMillis), math.MaxFloat64, 0, limit)
		if perr != nil {
			return nil, errorx.Wrap(ctx, perr, errorx.NewMsg("查询关注者发布列表失败"))
		}
		res := make([]followeeContent, 0, len(pairs))
		for _, p := range pairs {
			id, e := strconv.ParseInt(p.Key, 10, 64)
			if e != nil || id <= 0 {
				continue
			}
			res = append(res, followeeContent{id: id, publishedAt: int64(p.Score)})
		}
		return res, nil
	}

	rows, derr := contentRepo.ListPublishedByAuthorWithinWindow(followeeID, cutoffMillis, limit)
	if derr != nil {
		return nil, errorx.Wrap(ctx, derr, errorx.NewMsg("查询关注者发布内容失败"))
	}
	res := make([]followeeContent, 0, len(rows))
	for _, r := range rows {
		if r == nil || r.PublishedAt == nil {
			continue
		}
		res = append(res, followeeContent{id: r.ID, publishedAt: r.PublishedAt.UnixMilli()})
	}
	return res, nil
}
