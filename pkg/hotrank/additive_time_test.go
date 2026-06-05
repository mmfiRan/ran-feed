package hotrank

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func defaultAdditive() AdditiveTime {
	return AdditiveTime{
		Weights:       DefaultWeights(),
		HalfLifeHours: 24,
	}
}

func TestAdditive_Weighted(t *testing.T) {
	a := defaultAdditive()
	// 1 乘 like 加 3 乘 comment 加 4 乘 favorite
	assert.InDelta(t, 1*1+2*3+3*4, a.Weighted(1, 2, 3), 1e-9)
}

func TestAdditive_ZeroInteractionClampedToLog1(t *testing.T) {
	a := defaultAdditive()
	now := time.Now()
	// 0 互动 加权 clamp 到 1 log10(1) 等于 0 分值完全由时间项决定
	score := a.Score(0, now)
	expected := float64(now.Unix()) / a.timeDivisorSeconds()
	assert.InDelta(t, expected, score, 1e-3)
}

func TestAdditive_NewerPublishOutranksOlderAtZeroInteraction(t *testing.T) {
	a := defaultAdditive()
	now := time.Now()
	newer := a.Score(0, now)
	older := a.Score(0, now.Add(-48*time.Hour))
	// 冷启动自带解药 同为 0 互动 新内容时间项更大
	assert.Greater(t, newer, older)
}

func TestAdditive_TenXInteractionEqualsOneUnitTime(t *testing.T) {
	a := defaultAdditive()
	pub := time.Unix(1_700_000_000, 0)
	// log10 性质 互动量乘 10 log 项加 1.0
	low := a.Score(10, pub)
	high := a.Score(100, pub)
	assert.InDelta(t, 1.0, high-low, 1e-3)
}

func TestAdditive_HalfLifeCatchUp(t *testing.T) {
	a := defaultAdditive()
	// 设计第2节 晚发一个半衰期 S 乘 log10(2) 等于 halfLifeSeconds 约等于需 10 倍互动追平
	// 时间项相差 halfLifeSeconds 除以 S 等于 log10(2) 恰好等于互动量乘 10 的 log 增量的 log10(2) 倍
	s := a.timeDivisorSeconds()
	older := time.Unix(1_700_000_000, 0)
	newer := older.Add(time.Duration(a.HalfLifeHours) * time.Hour)
	gap := float64(newer.Unix()-older.Unix()) / s
	assert.InDelta(t, math.Log10(2), gap, 1e-6)
}

func TestAdditive_ZeroHalfLifeDropsTimeTerm(t *testing.T) {
	a := AdditiveTime{Weights: DefaultWeights(), HalfLifeHours: 0}
	// 半衰期为 0 退化为纯 log10 加权 无时间项
	score := a.Score(100, time.Now())
	assert.InDelta(t, 2.0, score, 1e-9)
}

func TestAdditive_MonotonicInWeighted(t *testing.T) {
	a := defaultAdditive()
	pub := time.Unix(1_700_000_000, 0)
	prev := a.Score(1, pub)
	for _, w := range []float64{2, 10, 100, 1000} {
		cur := a.Score(w, pub)
		assert.Greater(t, cur, prev)
		prev = cur
	}
}
