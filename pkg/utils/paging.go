package utils

type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

const (
	// DefaultPageSize 统一分页默认条数
	DefaultPageSize = 20
	// MaxPageSize 统一分页上限
	MaxPageSize = 50
)

// ClampPageSize 归一每页条数 size<=0 取 DefaultPageSize 超 MaxPageSize 取 MaxPageSize
func ClampPageSize[T Integer](size T) int {
	s := int(size)
	if s <= 0 {
		s = DefaultPageSize
	}
	if s > MaxPageSize {
		s = MaxPageSize
	}
	return s
}

// NormalizePage offset分页 返回归一后的 offset 与 limit
func NormalizePage[T Integer, S Integer](page T, pageSize S) (offset, limit int) {
	limit = ClampPageSize(pageSize)
	p := int(page)
	if p < 1 {
		p = 1
	}
	return (p - 1) * limit, limit
}
