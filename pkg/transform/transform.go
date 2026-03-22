package transform

import (
	"encoding/json"
	"fmt"
	"strings"
)

func DataTransform[T any](t T, data any) (T, error) {
	marshal, err := json.Marshal(data)
	if err != nil {
		return t, err
	}
	if err = json.Unmarshal(marshal, &t); err != nil {
		return t, err
	}
	return t, nil
}

// ParseEnum 通用枚举解析函数

func ParseEnum[T ~int32](valueMap map[string]int32, input string) (T, error) {
	upperInput := strings.ToUpper(input)
	if value, ok := valueMap[upperInput]; ok {
		return T(value), nil
	}

	var validValues []string
	for key := range valueMap {
		if !strings.HasSuffix(key, "_UNKNOWN") && key != "UNKNOWN" {
			validValues = append(validValues, key)
		}
	}
	var zero T
	return zero, fmt.Errorf("枚举值%s不在有效值范围%s内", input, strings.Join(validValues, ","))
}
