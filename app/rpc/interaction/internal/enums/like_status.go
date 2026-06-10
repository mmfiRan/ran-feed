// Package enums 业务专属枚举集中目录
package enums

import "fmt"

// LikeStatus 点赞状态 ran_feed_like 表的 status 字段类型
// 业务专属 不与其他状态机共用类型 避免误用
type LikeStatus int32

const (
	LikeStatusLike   LikeStatus = 10
	LikeStatusCancel LikeStatus = 20
)

// likeStatusNames 集中维护值与名称的映射 新增值只需在此处补充
var likeStatusNames = map[LikeStatus]string{
	LikeStatusLike:   "LIKE",
	LikeStatusCancel: "CANCEL",
}

func (s LikeStatus) Int32() int32 {
	return int32(s)
}

func (s LikeStatus) Valid() bool {
	_, ok := likeStatusNames[s]
	return ok
}

func (s LikeStatus) String() string {
	if name, ok := likeStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("LikeStatus(%d)", s)
}

// IsLiked 是否为点赞态
func (s LikeStatus) IsLiked() bool {
	return s == LikeStatusLike
}

// IsCancelled 是否为取消态
func (s LikeStatus) IsCancelled() bool {
	return s == LikeStatusCancel
}
