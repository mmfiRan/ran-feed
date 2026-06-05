package hotrank

import (
	"math"
	"time"
)

// AdditiveTime 热榜分等于 log10 加权取下限 1 加 发布秒 除以 S
//
//	log10 抗霸榜 头部边际递减 抗刷量 刷分代价指数级
//	时间为加法常数项 发布时刻即钉死 0 互动也能靠时间项进 TopN 冷启动自带解药
//	S 等于半衰期秒数 除以 log10(2) 控制晚发多久约等于需 10 倍互动追平
type AdditiveTime struct {
	Weights       Weights
	HalfLifeHours float64
}

// scorePrecision 分值保留小数位 时间项 1 秒约 3.5e-6 6 位小数足以区分秒级先后
const scorePrecision = 1e6

// timeDivisorSeconds 返回时间项分母 S 半衰期非正时退化为无时间项
func (a AdditiveTime) timeDivisorSeconds() float64 {
	if a.HalfLifeHours <= 0 {
		return 0
	}
	return a.HalfLifeHours * 3600 / math.Log10(2)
}

// Weighted 加权互动量等于 w_like 乘点赞 加 w_comment 乘评论 加 w_fav 乘收藏
func (a AdditiveTime) Weighted(likeCount, commentCount, favoriteCount int64) float64 {
	return float64(likeCount)*a.Weights.Like +
		float64(commentCount)*a.Weights.Comment +
		float64(favoriteCount)*a.Weights.Favorite
}

// Score 计算热榜分 log10 加权取下限 1 加 发布秒 除以 S
func (a AdditiveTime) Score(weighted float64, publishedAt time.Time) float64 {
	if weighted < 1 {
		weighted = 1
	}
	score := math.Log10(weighted)
	if s := a.timeDivisorSeconds(); s > 0 {
		score += float64(publishedAt.Unix()) / s
	}
	return math.Round(score*scorePrecision) / scorePrecision
}
