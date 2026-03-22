package xxljob

import (
	"encoding/json"
	"time"
)

const (
	SuccessCode = 200
	FailCode    = 500
)

const (
	HeaderAccessToken = "XXL-JOB-ACCESS-TOKEN"
	AltHeaderToken    = "Xxl-Job-Access-Token"
)

type ReturnT struct {
	Code    int         `json:"code"`
	Msg     string      `json:"msg"`
	Content interface{} `json:"content,omitempty"`
}

type returnTRaw struct {
	Code    int             `json:"code"`
	Msg     string          `json:"msg"`
	Content json.RawMessage `json:"content"`
}

func Ok(msg string) ReturnT {
	return ReturnT{
		Code: SuccessCode,
		Msg:  msg,
	}
}

func Fail(msg string) ReturnT {
	return ReturnT{
		Code: FailCode,
		Msg:  msg,
	}
}

// TriggerParam 是 XXL-Job admin 调用执行器 /run 的参数。
// Raw 保留原始 JSON，方便兼容未知字段。
type TriggerParam struct {
	JobID                 int64           `json:"jobId"`
	ExecutorHandler       string          `json:"executorHandler"`
	ExecutorParams        string          `json:"executorParams"`
	ExecutorBlockStrategy string          `json:"executorBlockStrategy"`
	ExecutorTimeout       int             `json:"executorTimeout"`
	LogID                 int64           `json:"logId"`
	LogDateTime           int64           `json:"logDateTime"`
	GlueType              string          `json:"glueType"`
	GlueSource            string          `json:"glueSource"`
	GlueUpdatetime        int64           `json:"glueUpdatetime"`
	BroadcastIndex        int             `json:"broadcastIndex"`
	BroadcastTotal        int             `json:"broadcastTotal"`
	Raw                   json.RawMessage `json:"-"`
}

// IdleBeatParam 是 /idleBeat 的参数。
type IdleBeatParam struct {
	JobID           int64  `json:"jobId"`
	ExecutorHandler string `json:"executorHandler"`
}

// KillParam 是 /kill 的参数。
type KillParam struct {
	JobID           int64  `json:"jobId"`
	ExecutorHandler string `json:"executorHandler"`
}

// LogParam 是 /log 的参数。
type LogParam struct {
	LogDateTime int64 `json:"logDateTim"`
	LogID       int64 `json:"logId"`
	FromLineNum int   `json:"fromLineNum"`
}

// LogResult 是 /log 的响应内容。
type LogResult struct {
	LogContent  string `json:"logContent"`
	FromLineNum int    `json:"fromLineNum"`
	ToLineNum   int    `json:"toLineNum"`
	IsEnd       bool   `json:"isEnd"`
}

// HandleCallbackParam 是 admin /api/callback 的参数。
type HandleCallbackParam struct {
	LogID       int64  `json:"logId"`
	LogDateTime int64  `json:"logDateTim"`
	HandleCode  int    `json:"handleCode"`
	HandleMsg   string `json:"handleMsg"`
}

func (p HandleCallbackParam) MarshalJSON() ([]byte, error) {
	type alias HandleCallbackParam
	return json.Marshal(struct {
		alias
		LogDateTime int64 `json:"logDateTime,omitempty"`
	}{
		alias:       alias(p),
		LogDateTime: p.LogDateTime,
	})
}

// RegistryParam 是 admin /api/registry 和 /api/registryRemove 的参数。
type RegistryParam struct {
	RegistryGroup string `json:"registryGroup"`
	RegistryKey   string `json:"registryKey"`
	RegistryValue string `json:"registryValue"`
}

// RunResult 是内部执行结果结构（用于回调）。
type RunResult struct {
	Code    int
	Msg     string
	EndTime time.Time
}

// BlockStrategy 是执行器阻塞策略。
type BlockStrategy string

const (
	BlockSerial  BlockStrategy = "SERIAL_EXECUTION"
	BlockDiscard BlockStrategy = "DISCARD_LATER"
	BlockCover   BlockStrategy = "COVER_EARLY"
)
