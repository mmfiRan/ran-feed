package contentservicelogic

import (
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/pkg/snowflake"
)

func buildContentModel(userID int64, contentType contentEnum.ContentTypeEnum, status contentEnum.ContentStatusEnum, visibility contentEnum.VisibilityEnum) *model.RanFeedContent {
	return &model.RanFeedContent{
		ID:          snowflake.GenID(),
		UserID:      userID,
		ContentType: contentType.Int32(),
		Status:      status.Int32(),
		Visibility:  visibility.Int32(),
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}
}
