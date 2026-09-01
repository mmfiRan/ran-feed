package enums

import "fmt"

// TranscodeStatusEnum 视频转码状态 10=未开始 20=处理中 30=成功 40=失败
type TranscodeStatusEnum int32

const (
	TranscodeStatusUnknown    TranscodeStatusEnum = 0
	TranscodeStatusPending    TranscodeStatusEnum = 10
	TranscodeStatusProcessing TranscodeStatusEnum = 20
	TranscodeStatusSuccess    TranscodeStatusEnum = 30
	TranscodeStatusFailed     TranscodeStatusEnum = 40
)

var transcodeStatusNames = map[TranscodeStatusEnum]string{
	TranscodeStatusUnknown:    "UNKNOWN",
	TranscodeStatusPending:    "PENDING",
	TranscodeStatusProcessing: "PROCESSING",
	TranscodeStatusSuccess:    "SUCCESS",
	TranscodeStatusFailed:     "FAILED",
}

var transcodeStatusMessages = map[TranscodeStatusEnum]string{
	TranscodeStatusUnknown:    "未知",
	TranscodeStatusPending:    "未开始",
	TranscodeStatusProcessing: "处理中",
	TranscodeStatusSuccess:    "成功",
	TranscodeStatusFailed:     "失败",
}

func (s TranscodeStatusEnum) Int32() int32 {
	return int32(s)
}

func (s TranscodeStatusEnum) Valid() bool {
	_, ok := transcodeStatusNames[s]
	return ok
}

func (s TranscodeStatusEnum) String() string {
	if name, ok := transcodeStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("TranscodeStatusEnum(%d)", s)
}

func (s TranscodeStatusEnum) Message() string {
	if msg, ok := transcodeStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}
