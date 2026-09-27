package contentresolver

import "ran-feed/app/rpc/content/internal/common/component/contentcache"

// Entry 一条内容的解析结果 缓存里的详情加上旁挂的作者与点赞 不带 protobuf 各调用方自己映射成各自的响应类型
type Entry struct {
	Detail       *contentcache.Detail
	AuthorName   string
	AuthorAvatar string
	IsLiked      bool
	LikeCount    int64
}
