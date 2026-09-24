package enums

import "fmt"

// ReviewDecisionEnum 审核决策 ran_feed_content_review 表的 decision 字段类型
// 审核通过拒绝与管理端下架恢复都落该表 各占一个决策值便于查历史
type ReviewDecisionEnum int32

const (
	ReviewDecisionUnknown   ReviewDecisionEnum = 0
	ReviewDecisionApprove   ReviewDecisionEnum = 10
	ReviewDecisionReject    ReviewDecisionEnum = 20
	ReviewDecisionTakenDown ReviewDecisionEnum = 30
	ReviewDecisionRestored  ReviewDecisionEnum = 40
)

var reviewDecisionNames = map[ReviewDecisionEnum]string{
	ReviewDecisionUnknown:   "UNKNOWN",
	ReviewDecisionApprove:   "APPROVE",
	ReviewDecisionReject:    "REJECT",
	ReviewDecisionTakenDown: "TAKEN_DOWN",
	ReviewDecisionRestored:  "RESTORED",
}

var reviewDecisionMessages = map[ReviewDecisionEnum]string{
	ReviewDecisionUnknown:   "未知",
	ReviewDecisionApprove:   "通过",
	ReviewDecisionReject:    "拒绝",
	ReviewDecisionTakenDown: "下架",
	ReviewDecisionRestored:  "恢复",
}

func (d ReviewDecisionEnum) Int32() int32 {
	return int32(d)
}

func (d ReviewDecisionEnum) Valid() bool {
	_, ok := reviewDecisionNames[d]
	return ok
}

func (d ReviewDecisionEnum) String() string {
	if name, ok := reviewDecisionNames[d]; ok {
		return name
	}
	return fmt.Sprintf("ReviewDecisionEnum(%d)", d)
}

func (d ReviewDecisionEnum) Message() string {
	if msg, ok := reviewDecisionMessages[d]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", d)
}

// IsReject 是否为审核拒绝 取拒绝理由时用
func (d ReviewDecisionEnum) IsReject() bool {
	return d == ReviewDecisionReject
}
