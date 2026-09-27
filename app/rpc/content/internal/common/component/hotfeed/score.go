package hotfeed

import (
	"context"
	"fmt"
	"math"
	"time"

	"ran-feed/app/rpc/count/count"
	"ran-feed/pkg/hotrank"
)

// calcScore 算分 log10 加权互动 加 发布秒 除以 S counts 为 nil 时按 0 互动
func calcScore(calculator hotrank.AdditiveTime, publishedAt time.Time, counts *count.ContentCountsItem) float64 {
	weighted := calculator.Weighted(counts.GetLikeCount(), counts.GetCommentCount(), counts.GetFavoriteCount())
	return calculator.Score(weighted, publishedAt)
}

func mergeWeights(w *hotrank.Weights) hotrank.Weights {
	base := hotrank.DefaultWeights()
	if w == nil {
		return base
	}
	if w.Like > 0 {
		base.Like = w.Like
	}
	if w.Comment > 0 {
		base.Comment = w.Comment
	}
	if w.Favorite > 0 {
		base.Favorite = w.Favorite
	}
	return base
}

// deriveWindowDays 从半衰期推导全量窗口 与加法时间模型的竞争视界对齐
// 窗口天数 等于 ceil windowDecades 乘 halfLifeHours 除以 24 乘 log10 2
// 半衰期非正时退化为固定兜底窗口 推导值封顶 maxWindowDays 防全表扫描
func deriveWindowDays(halfLifeHours float64) int {
	if halfLifeHours <= 0 {
		return defaultWindowDays
	}
	days := int(math.Ceil(windowDecades * halfLifeHours / (24 * math.Log10(2))))
	if days < 1 {
		days = 1
	}
	if days > maxWindowDays {
		days = maxWindowDays
	}
	return days
}

// batchGetCounts 批量回查内容互动计数 由 count 服务提供
func (f *Feed) batchGetCounts(ctx context.Context, contentIDs []int64) (map[int64]*count.ContentCountsItem, error) {
	countsByID := make(map[int64]*count.ContentCountsItem, len(contentIDs))
	if len(contentIDs) == 0 {
		return countsByID, nil
	}

	resp, err := f.countRpc.BatchGetContentCounts(ctx, &count.BatchGetContentCountsReq{
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

// batchUpdateHotScore 分批落库 hot_score 与 updated_at
func (f *Feed) batchUpdateHotScore(ctx context.Context, ids []int64, scores []float64, batchSize int) error {
	if len(ids) == 0 {
		return nil
	}
	if len(ids) != len(scores) {
		return fmt.Errorf("ids and scores length mismatch")
	}
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	for start := 0; start < len(ids); start += batchSize {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		if err := f.contentRepo.BatchUpdateHotScores(ctx, ids[start:end], scores[start:end], time.Now()); err != nil {
			return err
		}
	}
	return nil
}
