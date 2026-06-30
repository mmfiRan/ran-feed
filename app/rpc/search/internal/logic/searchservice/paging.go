package searchservicelogic

// 分页默认与上限 size 兜底 10 上限 50 page 从 1 起
const (
	defaultPageSize = 10
	maxPageSize     = 50
)

// pageToFromSize 把 page/size 归一化为 ES from/size
func pageToFromSize(page, size int32) (from int, limit int) {
	if size <= 0 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	if page < 1 {
		page = 1
	}
	return int((page - 1) * size), int(size)
}
