package hotrank

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func defaultDecay() ExpDecay {
	return ExpDecay{
		Weights:       DefaultWeights(),
		HalfLifeHours: 24,
	}
}

func TestDeltaScore_ZeroCount(t *testing.T) {
	d := defaultDecay()
	score := d.DeltaScore(Event{Action: ActionLike, Count: 0, EventTime: time.Now()}, time.Now())
	assert.Equal(t, 0.0, score)
}

func TestDeltaScore_NegativeCount(t *testing.T) {
	d := defaultDecay()
	score := d.DeltaScore(Event{Action: ActionLike, Count: -1, EventTime: time.Now()}, time.Now())
	assert.Equal(t, 0.0, score)
}

func TestDeltaScore_UnknownAction(t *testing.T) {
	d := defaultDecay()
	score := d.DeltaScore(Event{Action: "share", Count: 10, EventTime: time.Now()}, time.Now())
	assert.Equal(t, 0.0, score)
}

func TestDeltaScore_FutureEventTime(t *testing.T) {
	d := defaultDecay()
	now := time.Now()
	// 未来事件：ageHours 被 clamp 到 0，衰减因子 = 1
	future := now.Add(2 * time.Hour)
	score := d.DeltaScore(Event{Action: ActionLike, Count: 1, EventTime: future}, now)
	expected := 1.0 * d.Weights.Like * 1.0
	assert.InDelta(t, expected, score, 1e-9)
}

func TestDeltaScore_NoDecayAtEventTime(t *testing.T) {
	d := defaultDecay()
	now := time.Now()
	// 事件刚发生：ageHours=0，decay=1
	score := d.DeltaScore(Event{Action: ActionLike, Count: 2, EventTime: now}, now)
	assert.InDelta(t, 2.0*d.Weights.Like, score, 1e-9)
}

func TestDeltaScore_HalfLifeDecay(t *testing.T) {
	d := defaultDecay() // halfLife=24h
	now := time.Now()
	eventTime := now.Add(-24 * time.Hour) // 1 个半衰期前
	score := d.DeltaScore(Event{Action: ActionLike, Count: 1, EventTime: eventTime}, now)
	expected := 1.0 * d.Weights.Like * 0.5
	assert.InDelta(t, expected, score, 1e-9)
}

func TestDeltaScore_DoubleHalfLifeDecay(t *testing.T) {
	d := defaultDecay()
	now := time.Now()
	eventTime := now.Add(-48 * time.Hour) // 2 个半衰期
	score := d.DeltaScore(Event{Action: ActionLike, Count: 1, EventTime: eventTime}, now)
	expected := 1.0 * d.Weights.Like * 0.25
	assert.InDelta(t, expected, score, 1e-9)
}

func TestDeltaScore_ActionWeights(t *testing.T) {
	d := defaultDecay()
	now := time.Now()
	event := Event{Count: 1, EventTime: now}

	event.Action = ActionLike
	like := d.DeltaScore(event, now)

	event.Action = ActionComment
	comment := d.DeltaScore(event, now)

	event.Action = ActionFavorite
	favorite := d.DeltaScore(event, now)

	// comment=3x like，favorite=4x like
	assert.InDelta(t, 3*like, comment, 1e-9)
	assert.InDelta(t, 4*like, favorite, 1e-9)
}

func TestDeltaScore_ZeroHalfLife(t *testing.T) {
	d := ExpDecay{Weights: DefaultWeights(), HalfLifeHours: 0}
	now := time.Now()
	// 半衰期为 0 时 lambda=0，无衰减
	score := d.DeltaScore(Event{Action: ActionLike, Count: 1, EventTime: now.Add(-100 * time.Hour)}, now)
	assert.InDelta(t, d.Weights.Like, score, 1e-9)
}

func TestDeltaScore_MultipleCount(t *testing.T) {
	d := defaultDecay()
	now := time.Now()
	eventTime := now.Add(-24 * time.Hour)
	lambda := math.Ln2 / d.HalfLifeHours
	decay := math.Exp(-lambda * 24)
	score := d.DeltaScore(Event{Action: ActionComment, Count: 5, EventTime: eventTime}, now)
	expected := float64(5) * d.Weights.Comment * decay
	assert.InDelta(t, expected, score, 1e-9)
}
