// Package enums search 服务级业务枚举 实现 pkg/enums.Enum 契约
package enums

import "fmt"

// HistoryStatusEnum 搜索历史状态 ran_feed_search_history.status 10=正常
type HistoryStatusEnum int32

const (
	HistoryStatusUnknown HistoryStatusEnum = 0
	HistoryStatusNormal  HistoryStatusEnum = 10
)

var historyStatusNames = map[HistoryStatusEnum]string{
	HistoryStatusUnknown: "UNKNOWN",
	HistoryStatusNormal:  "NORMAL",
}

var historyStatusMessages = map[HistoryStatusEnum]string{
	HistoryStatusUnknown: "未知",
	HistoryStatusNormal:  "正常",
}

func (s HistoryStatusEnum) Int32() int32 {
	return int32(s)
}

func (s HistoryStatusEnum) Valid() bool {
	_, ok := historyStatusNames[s]
	return ok
}

func (s HistoryStatusEnum) String() string {
	if name, ok := historyStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("HistoryStatusEnum(%d)", s)
}

func (s HistoryStatusEnum) Message() string {
	if msg, ok := historyStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}
