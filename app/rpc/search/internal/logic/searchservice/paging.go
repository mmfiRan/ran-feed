package searchservicelogic

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"ran-feed/app/rpc/search/internal/es"
)

// 分页默认与上限 size 兜底 10 上限 50
const (
	defaultPageSize = 10
	maxPageSize     = 50
)

// normalizeSize 归一化每页条数
func normalizeSize(size int32) int {
	if size <= 0 {
		return defaultPageSize
	}
	if size > maxPageSize {
		return maxPageSize
	}
	return int(size)
}

// encodeCursor 把 ES 命中 sort 值数组编码为不透明游标 空数组返回空串
func encodeCursor(sortValues []any) string {
	if len(sortValues) == 0 {
		return ""
	}
	raw, err := json.Marshal(sortValues)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

// decodeCursor 解码不透明游标为 search_after 值数组 空串返回 nil 非法游标报错
func decodeCursor(cursor string) ([]any, error) {
	if cursor == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("非法游标 %w", err)
	}
	var values []any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("非法游标 %w", err)
	}
	return values, nil
}

// nextCursor 取末条命中 sort 值编码为下一页游标 命中不足一页表示已到底返回空串
func nextCursor(hits []es.Hit, size int) string {
	if len(hits) < size || len(hits) == 0 {
		return ""
	}
	return encodeCursor(hits[len(hits)-1].Sort)
}
