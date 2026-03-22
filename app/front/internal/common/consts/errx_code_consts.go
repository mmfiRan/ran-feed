package consts

import "ran-feed/pkg/errorx"

var (
	ErrUserNotLogin = errorx.New("用户未登录", 100101)
)
