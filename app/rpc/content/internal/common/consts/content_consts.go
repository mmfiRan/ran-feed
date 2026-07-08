package consts

const (
	Aliyun = "aliyun"
)

// 视频转码状态
const (
	// TranscodeStatusPending 转码未开始（发布时的占位值，等待异步转码任务接管）
	TranscodeStatusPending int32 = 10
)

// 管理端内容列表分页
const (
	AdminListDefaultPageSize = 20
	AdminListMaxPageSize     = 100
)
