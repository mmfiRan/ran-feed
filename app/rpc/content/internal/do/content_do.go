package do

import "time"

type ContentDO struct {
	ID          int64
	UserID      int64
	ContentType int32
	Status      int32
	Visibility  int32
	Version     int32
	IsDeleted   int32
	CreatedBy   int64
	UpdatedBy   int64
	PublishedAt *time.Time
}

type ArticleDO struct {
	ID          int64
	ContentID   int64
	Title       string
	Description *string
	Cover       string
	Content     string
	Version     int32
	IsDeleted   int32
}

type VideoDO struct {
	ID              int64
	ContentID       int64
	MediaID         int64
	OriginURL       string
	CoverURL        string
	Duration        int32
	TranscodeStatus int32
}

// ContentReviewDO 内容审核记录 created_by/updated_by 落审核管理员 id
type ContentReviewDO struct {
	ID        int64
	ContentID int64
	Decision  int32
	Reason    string
	CreatedBy int64
	UpdatedBy int64
}

// ContentDetailDO feed 二级缓存按 content_id 存的内容本征详情
type ContentDetailDO struct {
	ContentID   int64  `json:"content_id"`
	ContentType int32  `json:"content_type"`
	AuthorID    int64  `json:"author_id"`
	Title       string `json:"title"`
	CoverURL    string `json:"cover_url"`
	PublishedAt int64  `json:"published_at"`
	Visibility  int32  `json:"visibility"`
}
