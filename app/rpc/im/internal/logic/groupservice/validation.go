package groupservicelogic

import (
	"strings"
	"unicode/utf8"

	"ran-feed/pkg/errorx"
)

const (
	// maxGroupNameRunes 群名称最大字符数,与 DDL VARCHAR(64) 对齐
	maxGroupNameRunes = 64
	// maxGroupAvatarRunes 群头像最大字符数,与 DDL VARCHAR(512) 对齐
	maxGroupAvatarRunes = 512
)

// validateGroupMeta 校验并归一化群名称与头像
func validateGroupMeta(name, avatar string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxGroupNameRunes {
		return "", errorx.NewMsg("群名称非法")
	}
	if utf8.RuneCountInString(avatar) > maxGroupAvatarRunes {
		return "", errorx.NewMsg("群头像过长")
	}
	return name, nil
}

// sanitizeMemberIDs 去重并按原顺序过滤群主,非法 ID 报错
func sanitizeMemberIDs(ownerID int64, memberIDs []int64) ([]int64, error) {
	if len(memberIDs) == 0 {
		return nil, nil
	}
	seen := make(map[int64]struct{}, len(memberIDs))
	out := make([]int64, 0, len(memberIDs))
	for _, userID := range memberIDs {
		if userID <= 0 {
			return nil, errorx.NewMsg("成员ID非法")
		}
		if userID == ownerID {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		out = append(out, userID)
	}
	return out, nil
}
