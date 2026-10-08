package feedservicelogic

import (
	"sync"

	"ran-feed/app/rpc/content/internal/common/component/publishbox"
	contentconsts "ran-feed/app/rpc/content/internal/common/consts"

	"github.com/zeromicro/go-zero/core/mr"
)

// pullConcurrency 拉侧并发查作者发件箱的并发度 工程常量不下发配置
const pullConcurrency = 16

// fetchPullItems 并发查拉侧作者发件箱当前窗口 返回所有命中的候选并集与任意源是否还有更多
// 重复项交给 mergeScored 去重 单个作者查失败只记日志 不让一个大 V 拖垮整条流
func (l *FollowFeedLogic) fetchPullItems(pullAuthors []int64, cursorScore int64, pageSize int) ([]scoredID, bool) {
	if len(pullAuthors) == 0 {
		return nil, false
	}

	var (
		mu         sync.Mutex
		pool       = make([]scoredID, 0, len(pullAuthors)*pageSize)
		anyHasMore bool
		logger     = l.Logger
	)
	cutoff := contentconsts.WindowCutoffMillis()

	mr.ForEach(func(source chan<- int64) {
		for _, uid := range pullAuthors {
			source <- uid
		}
	}, func(uid int64) {
		items, hasMore, err := l.publishBox.QueryWindow(l.ctx, uid, cutoff, cursorScore, pageSize)
		if err != nil {
			logger.Errorf("查询拉侧发件箱失败 uid=%d err=%v", uid, err)
			return
		}
		if len(items) == 0 && !hasMore {
			return
		}
		mu.Lock()
		pool = append(pool, publishBoxScored(items)...)
		if hasMore {
			anyHasMore = true
		}
		mu.Unlock()
	}, mr.WithWorkers(pullConcurrency))

	return pool, anyHasMore
}

// publishBoxScored 把发件箱候选转本包 scoredID
func publishBoxScored(items []publishbox.ScoredID) []scoredID {
	out := make([]scoredID, 0, len(items))
	for _, it := range items {
		out = append(out, scoredID{id: it.ID, score: it.Score})
	}
	return out
}
