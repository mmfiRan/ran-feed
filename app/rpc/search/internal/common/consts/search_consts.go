package consts

// 内容类型 与 content 域对齐
const (
	ContentTypeArticle int32 = 10
	ContentTypeVideo   int32 = 20
)

// 可索引判活口径 内容仅已发布且公开 用户仅正常状态
const (
	ContentStatusPublished  int32 = 30
	ContentVisibilityPublic int32 = 10
	UserStatusNormal        int32 = 10
)
