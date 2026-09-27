package contentcache

// Detail 内容详情的二级缓存值 由 content article video 三张表拼出来
// 字段标签就是缓存里的 JSON 形状 改它等于改协议 要和 encodeForCache 一起看
type Detail struct {
	ContentID   int64  `json:"content_id"`
	ContentType int32  `json:"content_type"`
	AuthorID    int64  `json:"author_id"`
	Title       string `json:"title"`
	CoverURL    string `json:"cover_url"`
	PublishedAt int64  `json:"published_at"`
	Visibility  int32  `json:"visibility"`
}
