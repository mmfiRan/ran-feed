package enums

import "fmt"

// FavoriteStatusEnum 收藏状态 ran_feed_favorite 表的 status 字段类型
// 业务专属 不与其他状态机共用类型 避免误用
type FavoriteStatusEnum int32

const (
	// FavoriteStatusLegacy 早期写入未落 status 的历史数据 按已收藏处理
	FavoriteStatusLegacy FavoriteStatusEnum = 0
	FavoriteStatusActive FavoriteStatusEnum = 10
	FavoriteStatusCancel FavoriteStatusEnum = 20
)

// favoriteStatusNames 集中维护值与名称的映射 新增值只需在此处补充
var favoriteStatusNames = map[FavoriteStatusEnum]string{
	FavoriteStatusLegacy: "LEGACY",
	FavoriteStatusActive: "ACTIVE",
	FavoriteStatusCancel: "CANCEL",
}

var favoriteStatusMessages = map[FavoriteStatusEnum]string{
	FavoriteStatusLegacy: "历史数据",
	FavoriteStatusActive: "已收藏",
	FavoriteStatusCancel: "取消收藏",
}

func (s FavoriteStatusEnum) Int32() int32 {
	return int32(s)
}

func (s FavoriteStatusEnum) Valid() bool {
	_, ok := favoriteStatusNames[s]
	return ok
}

func (s FavoriteStatusEnum) String() string {
	if name, ok := favoriteStatusNames[s]; ok {
		return name
	}
	return fmt.Sprintf("FavoriteStatusEnum(%d)", s)
}

func (s FavoriteStatusEnum) Message() string {
	if msg, ok := favoriteStatusMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}

// IsActive 是否算已收藏 历史数据与正常态都算
func (s FavoriteStatusEnum) IsActive() bool {
	return s == FavoriteStatusLegacy || s == FavoriteStatusActive
}
