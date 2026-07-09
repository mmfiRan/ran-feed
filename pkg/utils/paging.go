package utils

// ClampPageSize 归一每页条数 size<=0 取 defaultSize 超 maxSize 取 maxSize maxSize<=0 表示不设上限
func ClampPageSize(size, defaultSize, maxSize int) int {
	if size <= 0 {
		size = defaultSize
	}
	if maxSize > 0 && size > maxSize {
		size = maxSize
	}
	return size
}

// NormalizePage 归一 offset 分页参数 页从1 pageSize 经 ClampPageSize 归一 返回 offset 与 limit
func NormalizePage(page, pageSize, defaultSize, maxSize int) (offset, limit int) {
	limit = ClampPageSize(pageSize, defaultSize, maxSize)
	if page < 1 {
		page = 1
	}
	return (page - 1) * limit, limit
}
