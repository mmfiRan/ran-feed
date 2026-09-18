package utils

import (
	"strconv"
	"strings"

	"ran-feed/app/admin/internal/types"
	"ran-feed/pkg/commonpb"
)

// ToEnumValue pb 枚举值转 HTTP 响应统一枚举结构
func ToEnumValue(v *commonpb.EnumValue) types.EnumValue {
	if v == nil {
		return types.EnumValue{}
	}
	return types.EnumValue{
		Code:    v.GetCode(),
		Name:    v.GetName(),
		Message: v.GetMessage(),
	}
}

// ParseInt64s 字符串 id 集合还原为 int64
func ParseInt64s(raw []string) []int64 {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(raw))
	out := make([]int64, 0, len(raw))
	for _, s := range raw {
		id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// FormatInt64s int64 id 集合转字符串集合
func FormatInt64s(ids []int64) []string {
	if ids == nil {
		return nil
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatInt(id, 10))
	}
	return out
}
