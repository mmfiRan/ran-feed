package hot_cold_update

import (
	"testing"
	"time"

	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/count/count"
	"ran-feed/pkg/hotrank"

	"github.com/stretchr/testify/assert"
)

func testCalculator() hotrank.AdditiveTime {
	return hotrank.AdditiveTime{
		Weights:       hotrank.DefaultWeights(),
		HalfLifeHours: defaultHalfLife,
	}
}

// TestCalcScore_UsesAllThreeCounts 三个互动字段任一增大都应抬高分值
// 防止漏读 点赞 评论 收藏 其中之一
func TestCalcScore_UsesAllThreeCounts(t *testing.T) {
	calculator := testCalculator()
	published := time.UnixMilli(1_700_000_000_000)
	now := published.Add(24 * time.Hour)

	base := &model.RanFeedContent{ID: 1, PublishedAt: &published}
	baseScore := calcScore(calculator, base, &count.ContentCountsItem{
		ContentId: 1, LikeCount: 10, CommentCount: 10, FavoriteCount: 10,
	}, now)

	cases := []struct {
		name   string
		counts *count.ContentCountsItem
	}{
		{"点赞增大", &count.ContentCountsItem{ContentId: 1, LikeCount: 100, CommentCount: 10, FavoriteCount: 10}},
		{"评论增大", &count.ContentCountsItem{ContentId: 1, LikeCount: 10, CommentCount: 100, FavoriteCount: 10}},
		{"收藏增大", &count.ContentCountsItem{ContentId: 1, LikeCount: 10, CommentCount: 10, FavoriteCount: 100}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := calcScore(calculator, base, tt.counts, now)
			assert.Greater(t, got, baseScore, "互动增多分值应更高")
		})
	}
}

// TestCalcScore_NilCountsEqualsZeroCounts nil 与全零应等价 都不 panic
func TestCalcScore_NilCountsEqualsZeroCounts(t *testing.T) {
	calculator := testCalculator()
	published := time.UnixMilli(1_700_000_000_000)
	now := published
	row := &model.RanFeedContent{ID: 1, PublishedAt: &published}

	fromNil := calcScore(calculator, row, nil, now)
	fromZero := calcScore(calculator, row, &count.ContentCountsItem{ContentId: 1}, now)

	assert.Equal(t, fromZero, fromNil)
}

// TestCalcScore_NilPublishedAtFallsBackToNow published_at 为空时以 now 兜底 不 panic
func TestCalcScore_NilPublishedAtFallsBackToNow(t *testing.T) {
	calculator := testCalculator()
	now := time.UnixMilli(1_700_000_000_000)
	counts := &count.ContentCountsItem{ContentId: 1, LikeCount: 5}

	fromNil := calcScore(calculator, &model.RanFeedContent{ID: 1}, counts, now)
	fromNow := calcScore(calculator, &model.RanFeedContent{ID: 1, PublishedAt: &now}, counts, now)

	assert.Equal(t, fromNow, fromNil)
}

// TestCalcScore_OutscoresOlderWithSameCounts 同互动量下 越新发布分值越高
func TestCalcScore_OutscoresOlderWithSameCounts(t *testing.T) {
	calculator := testCalculator()
	now := time.UnixMilli(1_700_000_000_000)
	older := now.Add(-72 * time.Hour)
	newer := now.Add(-1 * time.Hour)
	counts := &count.ContentCountsItem{ContentId: 1, LikeCount: 20, CommentCount: 5}

	olderScore := calcScore(calculator, &model.RanFeedContent{ID: 1, PublishedAt: &older}, counts, now)
	newerScore := calcScore(calculator, &model.RanFeedContent{ID: 2, PublishedAt: &newer}, counts, now)

	assert.Greater(t, newerScore, olderScore)
}
