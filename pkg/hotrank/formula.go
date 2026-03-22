package hotrank

import (
	"math"
	"time"
)

type Weights struct {
	Like     float64
	Comment  float64
	Favorite float64
}

// ExpDecay 基于指数衰减：score = raw * exp(-lambda * ageHours)
// halfLifeHours 表示半衰期（小时）
type ExpDecay struct {
	Weights       Weights
	HalfLifeHours float64
}

func DefaultWeights() Weights {
	return Weights{
		Like:     1,
		Comment:  3,
		Favorite: 4,
	}
}

func (d ExpDecay) DeltaScore(e Event, now time.Time) float64 {
	if e.Count <= 0 {
		return 0
	}
	weight := d.weight(e.Action)
	if weight == 0 {
		return 0
	}
	ageHours := now.Sub(e.EventTime).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	lambda := 0.0
	if d.HalfLifeHours > 0 {
		lambda = math.Ln2 / d.HalfLifeHours
	}
	decay := 1.0
	if lambda > 0 {
		decay = math.Exp(-lambda * ageHours)
	}
	return float64(e.Count) * weight * decay
}

func (d ExpDecay) weight(action ActionType) float64 {
	switch action {
	case ActionLike:
		return d.Weights.Like
	case ActionComment:
		return d.Weights.Comment
	case ActionFavorite:
		return d.Weights.Favorite
	default:
		return 0
	}
}
