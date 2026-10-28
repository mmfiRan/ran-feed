package consts

import "ran-feed/pkg/errorx"

// 客户端需要分支处理的业务错误
var (
	// ErrNotMember 非会话成员
	ErrNotMember = errorx.New("无权访问该会话", 100301)
	// ErrSendNotMember 非会话成员发送消息
	ErrSendNotMember = errorx.New("无权在该会话发送消息", 100301)
	// ErrRevokeWindow 超过可撤回时间窗口
	ErrRevokeWindow = errorx.New("超过可撤回时间", 100302)
	// ErrGroupFull 群成员已达上限
	ErrGroupFull = errorx.New("群成员已达上限", 100303)
	// ErrGroupDissolved 群已解散
	ErrGroupDissolved = errorx.New("群已解散", 100304)
)
