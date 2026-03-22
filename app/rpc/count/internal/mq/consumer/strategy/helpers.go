package strategy

import (
	"encoding/json"
	"strconv"
	"strings"
)

func isLikeActive(row map[string]interface{}) bool {
	status, ok := getInt64(row["status"])
	return ok && status == 10
}

func isCommentActive(row map[string]interface{}) bool {
	status, ok := getInt64(row["status"])
	if !ok || status != 10 {
		return false
	}

	isDeleted, ok := getInt64(row["is_deleted"])
	if !ok {
		isDeleted = 0
	}
	return isDeleted == 0
}

func isFollowActive(row map[string]interface{}) bool {
	status, ok := getInt64(row["status"])
	if !ok || status != 10 {
		return false
	}

	isDeleted, ok := getInt64(row["is_deleted"])
	if !ok {
		isDeleted = 0
	}
	return isDeleted == 0
}

func isContentDeletedTransition(row map[string]interface{}, oldRow map[string]interface{}) bool {
	if row == nil || oldRow == nil {
		return false
	}
	oldValRaw, ok := oldRow["is_deleted"]
	if !ok {
		return false
	}
	oldVal, okOld := getInt64(oldValRaw)
	newVal, okNew := getInt64(row["is_deleted"])
	if !okOld || !okNew {
		return false
	}
	return oldVal == 0 && newVal == 1
}

func mergedValue(newRow map[string]interface{}, oldRow map[string]interface{}, key string) interface{} {
	if oldRow != nil {
		if v, ok := oldRow[key]; ok {
			return v
		}
	}
	if newRow == nil {
		return nil
	}
	return newRow[key]
}

func boolDelta(before, after bool) int64 {
	if !before && after {
		return 1
	}
	if before && !after {
		return -1
	}
	return 0
}

func getInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case nil:
		return 0, false
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		return int64(n), true
	case float64:
		return int64(n), true
	case json.Number:
		val, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return val, true
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0, false
		}
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, false
		}
		return val, true
	default:
		return 0, false
	}
}
