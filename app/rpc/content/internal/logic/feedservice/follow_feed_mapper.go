package feedservicelogic

import (
	"sort"

	"ran-feed/app/rpc/content/internal/common/component/followfeed"
)

// followFeedScored 把收件箱候选转本包 scoredID
func followFeedScored(items []followfeed.ScoredID) []scoredID {
	out := make([]scoredID, 0, len(items))
	for _, it := range items {
		out = append(out, scoredID{id: it.ID, score: it.Score})
	}
	return out
}

// mergeScored 合并推侧收件箱与拉侧池 按复合游标(score id)过滤去重排序 截取 pageSize
// 各源 inclusive 取到游标分 边界同分在此按 (score id) 精确剔除 返回 content_id 列表 hasMore 复合游标
func mergeScored(push []scoredID, pushHasMore bool, pullPool []scoredID, pullHasMore bool, cursorScore, cursorID int64, pageSize int) ([]int64, bool, string) {
	seen := make(map[int64]struct{}, len(push)+len(pullPool))
	merged := make([]scoredID, 0, len(push)+len(pullPool))
	appendUniq := func(src []scoredID) {
		for _, s := range src {
			if s.id <= 0 || !afterCursor(s, cursorScore, cursorID) {
				continue
			}
			if _, ok := seen[s.id]; ok {
				continue
			}
			seen[s.id] = struct{}{}
			merged = append(merged, s)
		}
	}
	appendUniq(push)
	appendUniq(pullPool)

	// 按 published_at 倒序 同分以 content_id 倒序保证游标稳定
	sortScored(merged)

	overflow := len(merged) > pageSize
	if overflow {
		merged = merged[:pageSize]
	}
	hasMore := pushHasMore || pullHasMore || overflow

	nextCursor := ""
	if hasMore && len(merged) > 0 {
		last := merged[len(merged)-1]
		nextCursor = formatCursor(last.score, last.id)
	}
	return scoredIDsToIDs(merged), hasMore, nextCursor
}

// sortScored 按 published_at 倒序 同分以 content_id 倒序 保证游标稳定
func sortScored(items []scoredID) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].id > items[j].id
	})
}

// scoredIDsToIDs 按当前顺序抽出 content_id 丢弃 score
func scoredIDsToIDs(items []scoredID) []int64 {
	ids := make([]int64, 0, len(items))
	for _, s := range items {
		ids = append(ids, s.id)
	}
	return ids
}
