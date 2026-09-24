package hot_update

import (
	"testing"
	"time"

	"ran-feed/app/rpc/count/count"
	"ran-feed/pkg/hotrank"

	"github.com/stretchr/testify/assert"
)

func testCalculator() hotrank.AdditiveTime {
	return hotrank.AdditiveTime{
		Weights:       hotrank.DefaultWeights(),
		HalfLifeHours: defaultHalfLifeHour,
	}
}

// TestCalcScore_UsesAllThreeCounts 三个互动字段任一增大都应抬高分值 防漏读其一
func TestCalcScore_UsesAllThreeCounts(t *testing.T) {
	calculator := testCalculator()
	published := time.UnixMilli(1_700_000_000_000)
	base := calcScore(calculator, published, &count.ContentCountsItem{LikeCount: 10, CommentCount: 10, FavoriteCount: 10})

	cases := []struct {
		name   string
		counts *count.ContentCountsItem
	}{
		{"点赞增大", &count.ContentCountsItem{LikeCount: 100, CommentCount: 10, FavoriteCount: 10}},
		{"评论增大", &count.ContentCountsItem{LikeCount: 10, CommentCount: 100, FavoriteCount: 10}},
		{"收藏增大", &count.ContentCountsItem{LikeCount: 10, CommentCount: 10, FavoriteCount: 100}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert.Greater(t, calcScore(calculator, published, tt.counts), base)
		})
	}
}

// TestCalcScore_NilCountsEqualsZeroCounts nil 与全零应等价 都不 panic
func TestCalcScore_NilCountsEqualsZeroCounts(t *testing.T) {
	calculator := testCalculator()
	published := time.UnixMilli(1_700_000_000_000)

	assert.Equal(t, calcScore(calculator, published, &count.ContentCountsItem{}), calcScore(calculator, published, nil))
}

// TestCalcScore_OutscoresOlderWithSameCounts 同互动量下 越新发布分值越高
func TestCalcScore_OutscoresOlderWithSameCounts(t *testing.T) {
	calculator := testCalculator()
	newer := time.UnixMilli(1_700_000_000_000)
	older := newer.Add(-72 * time.Hour)
	counts := &count.ContentCountsItem{LikeCount: 20, CommentCount: 5}

	assert.Greater(t, calcScore(calculator, newer, counts), calcScore(calculator, older, counts))
}
