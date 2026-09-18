package utils

import (
	"errors"
	"strings"
)

const defaultCountryCode = "86"

// NormalizeMobile 把手机号化成E.164格式
func NormalizeMobile(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "").Replace(s)
	if s == "" {
		return "", errors.New("手机号不能为空")
	}

	switch {
	case strings.HasPrefix(s, "00"):
		s = "+" + strings.TrimPrefix(s, "00")
	case !strings.HasPrefix(s, "+"):
		s = "+" + defaultCountryCode + s
	}

	digits := strings.TrimPrefix(s, "+")
	if len(digits) < 8 || len(digits) > 15 {
		return "", errors.New("手机号长度不合法")
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return "", errors.New("手机号含非法字符")
		}
	}
	return s, nil
}
