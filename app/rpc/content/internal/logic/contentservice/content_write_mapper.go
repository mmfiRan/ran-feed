package contentservicelogic

import (
	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/pkg/snowflake"
)

// buildContentModel 四个建内容入口共用的主行构造 只填必要字段 热度 版本号 时间戳都交给库默认值
func buildContentModel(userID int64, contentType content.ContentType, status content.ContentStatus, visibility int32) *model.RanFeedContent {
	return &model.RanFeedContent{
		ID:          snowflake.GenID(),
		UserID:      userID,
		ContentType: int32(contentType),
		Status:      int32(status),
		Visibility:  visibility,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}
}
