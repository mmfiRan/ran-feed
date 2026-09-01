package enums

import "fmt"

// ContentStatusEnum 内容状态 10=草稿 20=处理中 30=已发布 40=失败 50=下架 60=待审 70=拒绝
type ContentStatusEnum int32

const (
	ContentStatusUnknown       ContentStatusEnum = 0
	ContentStatusDraft         ContentStatusEnum = 10
	ContentStatusProcessing    ContentStatusEnum = 20
	ContentStatusPublished     ContentStatusEnum = 30
	ContentStatusFailed        ContentStatusEnum = 40
	ContentStatusTakenDown     ContentStatusEnum = 50
	ContentStatusPendingReview ContentStatusEnum = 60
	ContentStatusRejected      ContentStatusEnum = 70
)

var contentStatusNames = map[ContentStatusEnum]string{
	ContentStatusUnknown:       "UNKNOWN",
	ContentStatusDraft:         "DRAFT",
	ContentStatusProcessing:    "PROCESSING",
	ContentStatusPublished:     "PUBLISHED",
	ContentStatusFailed:        "FAILED",
	ContentStatusTakenDown:     "TAKEN_DOWN",
	ContentStatusPendingReview: "PENDING_REVIEW",
	ContentStatusRejected:      "REJECTED",
}

var contentStatusMessages = map[ContentStatusEnum]string{
	ContentStatusUnknown:       "未知",
	ContentStatusDraft:         "草稿",
	ContentStatusProcessing:    "处理中",
	ContentStatusPublished:     "已发布",
	ContentStatusFailed:        "失败",
	ContentStatusTakenDown:     "已下架",
	ContentStatusPendingReview: "待审",
	ContentStatusRejected:      "已拒绝",
}

func (s ContentStatusEnum) Int32() int32 {
	return int32(s)
}

func (s ContentStatusEnum) Valid() bool {
	_, ok := contentStatusNames[s]
	return ok
}

func (s ContentStatusEnum) String() string {
	if name, ok := contentStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("ContentStatusEnum(%d)", s)
}

func (s ContentStatusEnum) Message() string {
	if msg, ok := contentStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}
