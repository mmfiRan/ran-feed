package consts

// 可索引判活口径 内容仅已发布且公开 用户仅正常状态 查询链路 ES filter 用
const (
	ContentStatusPublished  int32 = 30
	ContentVisibilityPublic int32 = 10
	UserStatusNormal        int32 = 10
)

// canal 源表名 消费者按表路由 search 作为 CDC 消费者需知道所订阅的源表
const (
	SourceTableContent = "ran_feed_content"
	SourceTableArticle = "ran_feed_article"
	SourceTableVideo   = "ran_feed_video"
	SourceTableUser    = "ran_feed_user"
)
