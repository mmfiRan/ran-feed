package enums

import "fmt"

// ReviewDecisionEnum 审核决策 10=通过 20=拒绝
type ReviewDecisionEnum int32

const (
	ReviewDecisionUnknown ReviewDecisionEnum = 0
	ReviewDecisionApprove ReviewDecisionEnum = 10
	ReviewDecisionReject  ReviewDecisionEnum = 20
)

var reviewDecisionNames = map[ReviewDecisionEnum]string{
	ReviewDecisionUnknown: "UNKNOWN",
	ReviewDecisionApprove: "APPROVE",
	ReviewDecisionReject:  "REJECT",
}

var reviewDecisionMessages = map[ReviewDecisionEnum]string{
	ReviewDecisionUnknown: "未知",
	ReviewDecisionApprove: "通过",
	ReviewDecisionReject:  "拒绝",
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
