package feedservicelogic

import (
	"strconv"
	"strings"
)

// scoredID feed zset 的成员 member 是 content_id score 是 published_at 毫秒
type scoredID struct {
	id    int64
	score int64
}

// parseCursor 解析 score:id 复合游标 解不出来或非法一律当从头开始
func parseCursor(cursor string) (int64, int64) {
	if cursor == "" || cursor == "0" {
		return 0, 0
	}
	if i := strings.IndexByte(cursor, ':'); i >= 0 {
		score, e1 := strconv.ParseInt(cursor[:i], 10, 64)
		id, e2 := strconv.ParseInt(cursor[i+1:], 10, 64)
		if e1 != nil || e2 != nil || score <= 0 {
			return 0, 0
		}
		if id < 0 {
			id = 0
		}
		return score, id
	}
	score, err := strconv.ParseInt(cursor, 10, 64)
	if err != nil || score <= 0 {
		return 0, 0
	}
	return score, 0
}

// formatCursor 组装 "score:id" 复合游标
func formatCursor(score, id int64) string {
	return strconv.FormatInt(score, 10) + ":" + strconv.FormatInt(id, 10)
}

// afterCursor 判断 s 是否严格排在游标之后 总序是 published_at 降序 同一毫秒再按 content_id 降序
func afterCursor(s scoredID, cursorScore, cursorID int64) bool {
	if cursorScore <= 0 {
		return true
	}
	if s.score != cursorScore {
		return s.score < cursorScore
	}
	return s.id < cursorID
}
