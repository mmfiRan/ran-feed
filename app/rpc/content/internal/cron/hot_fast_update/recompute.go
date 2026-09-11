package hot_fast_update

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"ran-feed/app/rpc/content/content"
	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/count/count"
	"ran-feed/pkg/hotrank"
)

// collectDirtyIDs 逐分片冻结脏集合并 SSCAN 收集脏 contentID
// 冻结脚本把活跃桶原子搬到处理中桶 处理期间新互动安全堆进活跃桶 双缓冲
func (j *HotFastUpdateJob) collectDirtyIDs(ctx context.Context, shards int) ([]int64, error) {
	seen := make(map[int64]struct{})
	ids := make([]int64, 0)
	for shard := 0; shard < shards; shard++ {
		activeKey := rediskey.BuildHotFeedDirtyKey(shard)
		procKey := rediskey.BuildHotFeedDirtyProcKey(shard)
		if _, err := j.svc.Redis.EvalCtx(ctx, luautils.FreezeHotDirtyScript, []string{activeKey, procKey}); err != nil {
			return nil, err
		}

		// 防止极端场景单个大key造成redis阻塞
		var cursor uint64
		for {
			members, next, err := j.svc.Redis.SscanCtx(ctx, procKey, cursor, "", defaultScanBatch)
			if err != nil {
				return nil, err
			}
			for _, m := range members {
				id, perr := strconv.ParseInt(m, 10, 64)
				if perr != nil || id <= 0 {
					continue
				}
				if _, ok := seen[id]; ok {
					continue
				}
				seen[id] = struct{}{}
				ids = append(ids, id)
			}
			cursor = next
			if cursor == 0 {
				break
			}
		}
	}
	return ids, nil
}

// recomputeAndOverwrite 对脏 ID 回查计数总量算全分 ZADD 覆盖主榜 已删或非公开的从主榜 ZREM
// 每批 ZADD 后立即把主榜裁回 mainN 候选池大小 主榜瞬时上限 = mainN + 批大小
// 避免脏数据量大时主榜一次性膨胀成大 key 及末尾一刀删大量成员阻塞 Redis
// 每条脏 ID 的新分都照写 含掉分内容 故无降分赖榜问题 裁剪只删分数最低的多余成员
func (j *HotFastUpdateJob) recomputeAndOverwrite(ctx context.Context, calculator hotrank.AdditiveTime, dirtyIDs []int64, mainN int) error {
	statusPublished := int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED)
	visibilityPublic := int32(content.Visibility_VISIBILITY_PUBLIC)

	for start := 0; start < len(dirtyIDs); start += defaultBatchSize {
		end := start + defaultBatchSize
		if end > len(dirtyIDs) {
			end = len(dirtyIDs)
		}
		batch := dirtyIDs[start:end]

		// 拿 published_at 加过滤状态 未返回的即已删或非公开 从主榜移除
		contentMap, err := j.contentRepo.BatchGetRecommendByIDs(statusPublished, visibilityPublic, batch)
		if err != nil {
			return fmt.Errorf("批量拉取推荐内容失败 %w", err)
		}

		staleMembers := make([]any, 0)
		validIDs := make([]int64, 0, len(batch))
		for _, id := range batch {
			if _, ok := contentMap[id]; ok {
				validIDs = append(validIDs, id)
			} else {
				staleMembers = append(staleMembers, strconv.FormatInt(id, 10))
			}
		}
		if len(staleMembers) > 0 {
			if _, err := j.svc.Redis.ZremCtx(ctx, rediskey.RedisFeedHotGlobalKey, staleMembers...); err != nil {
				return err
			}
		}
		if len(validIDs) == 0 {
			continue
		}

		// 回查计数总量 点赞 评论 收藏 算全分
		countsByID, err := j.batchGetCounts(ctx, validIDs)
		if err != nil {
			return fmt.Errorf("回查互动计数失败 %w", err)
		}

		dbIDs := make([]int64, 0, len(validIDs))
		dbScores := make([]float64, 0, len(validIDs))
		redisArgs := make([]interface{}, 0, len(validIDs)*2) // zadd 参数 score member 交替
		for _, id := range validIDs {
			row := contentMap[id]
			publishedAt := time.Now()
			if row.PublishedAt != nil {
				publishedAt = row.PublishedAt.UTC()
			}
			counts := countsByID[id]
			weighted := calculator.Weighted(counts.GetLikeCount(), counts.GetCommentCount(), counts.GetFavoriteCount())
			score := calculator.Score(weighted, publishedAt)
			// ZADD 覆盖非 ZINCRBY 分值是按总量重算的时点值 自愈漂移和丢事件
			redisArgs = append(redisArgs, score, strconv.FormatInt(id, 10))
			dbIDs = append(dbIDs, id)
			dbScores = append(dbScores, score)
		}

		// 整批一次 ZADD 替代逐条往返 与冷更共用重建脚本口径一致
		if len(redisArgs) > 0 {
			if _, err = j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotFeedZSetScript, []string{
				rediskey.RedisFeedHotGlobalKey,
			}, redisArgs...); err != nil {
				return err
			}
			// 分批裁剪 每批 ZADD 后立即裁回 mainN 主榜瞬时不超过 mainN+批大小
			// ZREMRANGEBYRANK 删除升序 rank [0,-(mainN+1)] 即最低分多余成员 成员数<=mainN 时空操作
			if mainN > 0 {
				if _, err = j.svc.Redis.ZremrangebyrankCtx(ctx, rediskey.RedisFeedHotGlobalKey, 0, -(int64(mainN) + 1)); err != nil {
					return err
				}
			}
		}

		// 同步落库 hot_score 与主榜口径一致
		if err = j.contentRepo.BatchUpdateHotScores(dbIDs, dbScores, time.Now()); err != nil {
			return fmt.Errorf("批量落库 hot_score 失败 %w", err)
		}
	}
	return nil
}

// batchGetCounts 批量回查内容的点赞 评论 收藏总量 由 count 服务提供
func (j *HotFastUpdateJob) batchGetCounts(ctx context.Context, contentIDs []int64) (map[int64]*count.ContentCountsItem, error) {
	countsByID := make(map[int64]*count.ContentCountsItem, len(contentIDs))
	if len(contentIDs) == 0 {
		return countsByID, nil
	}

	resp, err := j.svc.CountRpc.BatchGetContentCounts(ctx, &count.BatchGetContentCountsReq{
		ContentIds: contentIDs,
	})
	if err != nil {
		return nil, err
	}
	for _, item := range resp.GetItems() {
		if item == nil || item.GetContentId() <= 0 {
			continue
		}
		countsByID[item.GetContentId()] = item
	}
	return countsByID, nil
}
