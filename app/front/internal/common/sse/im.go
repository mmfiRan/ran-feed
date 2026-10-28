package sse

import (
	"encoding/json"
	"fmt"

	"ran-feed/pkg/event/imevent"
)

// IMTransformer im:push 载荷拆成目标用户与原始帧
func IMTransformer(raw []byte) ([]int64, []byte, error) {
	if len(raw) == 0 {
		return nil, nil, fmt.Errorf("空载荷")
	}
	var env imevent.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, nil, fmt.Errorf("解析 im push payload 失败: %w", err)
	}
	if len(env.UserIDs) == 0 {
		return nil, nil, fmt.Errorf("空接收者")
	}
	if len(env.Frame) == 0 || string(env.Frame) == "null" {
		return nil, nil, fmt.Errorf("空帧")
	}
	userIDs := make([]int64, 0, len(env.UserIDs))
	for _, userID := range env.UserIDs {
		if userID > 0 {
			userIDs = append(userIDs, userID)
		}
	}
	if len(userIDs) == 0 {
		return nil, nil, fmt.Errorf("非法接收者")
	}
	return userIDs, env.Frame, nil
}
