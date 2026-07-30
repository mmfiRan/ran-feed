package utils

const (
	// DefaultPageSize 统一分页默认条数
	DefaultPageSize = 20
	// MaxPageSize 统一分页上限 防脏输入放大回源
	MaxPageSize = 50
)

// ClampPageSize 归一每页条数 size<=0 取 DefaultPageSize 超 MaxPageSize 取 MaxPageSize
func ClampPageSize(size int) int {
	if size <= 0 {
		size = DefaultPageSize
	}
	if size > MaxPageSize {
		size = MaxPageSize
	}
	return size
}

// NormalizePage offset分页
func NormalizePage(page, pageSize int) (offset, limit int) {
	limit = ClampPageSize(pageSize)
	if page < 1 {
		page = 1
	}
	return (page - 1) * limit, limit
}
