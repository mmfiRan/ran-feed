package canal

import (
	"encoding/json"
	"fmt"
)

// unknownEventID eventID 缺失时的占位
const unknownEventID = "unknown"

// RowEventID 行级幂等标识 优先用行主键 缺失则用行内容加下标 保证同一行只处理一次
func RowEventID(eventID, table, op string, row map[string]any, idx int) string {
	if eventID == "" {
		eventID = unknownEventID
	}
	if rowID, ok := ParseInt64(row["id"]); ok && rowID > 0 {
		return sha1Hex(fmt.Sprintf("%s|%s|%s|%d", eventID, table, op, rowID))
	}

	rowJSON, _ := json.Marshal(row)
	return sha1Hex(fmt.Sprintf("%s|%s|%s|%d|%s", eventID, table, op, idx, string(rowJSON)))
}
