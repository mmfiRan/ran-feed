package hotrank

import "time"

type ActionType string

const (
	ActionLike     ActionType = "like"
	ActionComment  ActionType = "comment"
	ActionFavorite ActionType = "favorite"
)

type Event struct {
	Action    ActionType
	Count     int64
	EventTime time.Time
}

type Calculator interface {
	DeltaScore(e Event, now time.Time) float64
}
