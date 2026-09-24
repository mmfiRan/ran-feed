package hot_update

import (
	"context"
	"fmt"
	"strconv"
	"time"

	rediskey "ran-feed/app/rpc/content/internal/common/consts/redis"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	luautils "ran-feed/app/rpc/content/internal/common/utils/lua"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/pkg/hotrank"
)

// rebuildFromDB 按 id 倒序游标分页扫窗口内 已发布且公开 内容 时点重算写主榜并落库
// 直接写主榜不清空 与增量模式共用同一主榜 天然无重建空窗
func (j *Job) rebuildFromDB(ctx context.Context, calculator hotrank.AdditiveTime, startTime, now time.Time, p Params) error {
	cursorID := int64(0)
	for {
		rows, err := j.contentRepo.ListColdUpdateContents(
			contentEnum.ContentStatusPublished.Int32(),
			contentEnum.VisibilityPublic.Int32(),
			startTime,
			cursorID,
			p.PageSize,
		)
		if err != nil {
			return fmt.Errorf("分页拉取全量内容失败 %w", err)
		}
		if len(rows) == 0 {
			return nil
		}

		// 先筛出可算分的行 再批量回查互动计数 与增量模式同口径
		validRows := make([]*model.RanFeedContent, 0, len(rows))
		contentIDs := make([]int64, 0, len(rows))
		for _, row := range rows {
			if row == nil || row.PublishedAt == nil {
				continue
			}
			validRows = append(validRows, row)
			contentIDs = append(contentIDs, row.ID)
		}

		countsByID, err := j.batchGetCounts(ctx, contentIDs)
		if err != nil {
			return fmt.Errorf("批量回查互动计数失败 %w", err)
		}

		ids := make([]int64, 0, len(validRows))
		scores := make([]float64, 0, len(validRows))
		redisArgs := make([]interface{}, 0, len(validRows)*2)
		for _, row := range validRows {
			score := calcScore(calculator, row.PublishedAt.UTC(), countsByID[row.ID])
			ids = append(ids, row.ID)
			scores = append(scores, score)
			redisArgs = append(redisArgs, score, strconv.FormatInt(row.ID, 10))
		}

		if len(ids) > 0 {
			if err := j.batchUpdateHotScore(ctx, ids, scores, p.BatchSize); err != nil {
				return fmt.Errorf("批量落库 hot_score 失败 %w", err)
			}
			if _, err := j.svc.Redis.EvalCtx(ctx, luautils.RebuildHotFeedZSetScript, []string{
				rediskey.RedisFeedHotGlobalKey,
			}, redisArgs...); err != nil {
				return fmt.Errorf("写主榜失败 %w", err)
			}
		}

		// 下一页继续向更小 ID 扫描
		cursorID = rows[len(rows)-1].ID
		if len(rows) < p.PageSize {
			return nil
		}
	}
}
