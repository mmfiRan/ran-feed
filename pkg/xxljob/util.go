package xxljob

import (
	"strconv"
	"strings"
	"time"
)

func itoa(v int) string {
	return strconv.Itoa(v)
}

func timeTicker(d time.Duration) *time.Ticker {
	return time.NewTicker(d)
}

func shortenForLog(s string, limit int) string {
	s = strings.TrimSpace(s)
	if limit <= 0 || len(s) <= limit {
		return s
	}
	return s[:limit] + "...(truncated)"
}
