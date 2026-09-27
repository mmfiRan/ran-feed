package enums

import (
	"testing"

	"ran-feed/app/rpc/content/content"

	"github.com/stretchr/testify/assert"
)

// TestEnumValuesMatchProto 业务枚举取值必须与 proto 枚举一致
// 只覆盖有 proto 对应的枚举 TranscodeStatus 无 proto 定义不在此列
func TestEnumValuesMatchProto(t *testing.T) {
	tests := []struct {
		name  string
		biz   int32
		proto int32
	}{
		{"内容类型 未知", ContentTypeUnknown.Int32(), int32(content.ContentType_CONTENT_TYPE_UNSPECIFIED)},
		{"内容类型 文章", ContentTypeArticle.Int32(), int32(content.ContentType_CONTENT_TYPE_ARTICLE)},
		{"内容类型 视频", ContentTypeVideo.Int32(), int32(content.ContentType_CONTENT_TYPE_VIDEO)},

		{"内容状态 未知", ContentStatusUnknown.Int32(), int32(content.ContentStatus_CONTENT_STATUS_UNSPECIFIED)},
		{"内容状态 草稿", ContentStatusDraft.Int32(), int32(content.ContentStatus_CONTENT_STATUS_DRAFT)},
		{"内容状态 已发布", ContentStatusPublished.Int32(), int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED)},
		{"内容状态 失败", ContentStatusFailed.Int32(), int32(content.ContentStatus_CONTENT_STATUS_FAILED)},
		{"内容状态 下架", ContentStatusTakenDown.Int32(), int32(content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN)},
		{"内容状态 待审", ContentStatusPendingReview.Int32(), int32(content.ContentStatus_CONTENT_STATUS_PENDING_REVIEW)},
		{"内容状态 拒绝", ContentStatusRejected.Int32(), int32(content.ContentStatus_CONTENT_STATUS_REJECTED)},

		{"可见性 未知", VisibilityUnknown.Int32(), int32(content.Visibility_VISIBILITY_UNSPECIFIED)},
		{"可见性 公开", VisibilityPublic.Int32(), int32(content.Visibility_VISIBILITY_PUBLIC)},
		{"可见性 私密", VisibilityPrivate.Int32(), int32(content.Visibility_VISIBILITY_PRIVATE)},

		{"审核决策 未知", ReviewDecisionUnknown.Int32(), int32(content.ReviewDecision_REVIEW_DECISION_UNSPECIFIED)},
		{"审核决策 通过", ReviewDecisionApprove.Int32(), int32(content.ReviewDecision_REVIEW_DECISION_APPROVE)},
		{"审核决策 拒绝", ReviewDecisionReject.Int32(), int32(content.ReviewDecision_REVIEW_DECISION_REJECT)},
		{"审核决策 下架", ReviewDecisionTakenDown.Int32(), int32(content.ReviewDecision_REVIEW_DECISION_TAKEN_DOWN)},
		{"审核决策 恢复", ReviewDecisionRestored.Int32(), int32(content.ReviewDecision_REVIEW_DECISION_RESTORED)},

		{"上传场景 未知", UploadSceneUnknown.Int32(), int32(content.UploadScene_UPLOAD_SCENE_UNSPECIFIED)},
		{"上传场景 文章封面", UploadSceneArticleCover.Int32(), int32(content.UploadScene_UPLOAD_SCENE_ARTICLE_COVER)},
		{"上传场景 视频", UploadSceneVideo.Int32(), int32(content.UploadScene_UPLOAD_SCENE_VIDEO)},
		{"上传场景 头像", UploadSceneAvatar.Int32(), int32(content.UploadScene_UPLOAD_SCENE_AVATAR)},

		{"文件扩展名 未知", FileExtUnknown.Int32(), int32(content.FileExt_FILE_EXT_UNSPECIFIED)},
		{"文件扩展名 jpg", FileExtJpg.Int32(), int32(content.FileExt_FILE_EXT_JPG)},
		{"文件扩展名 png", FileExtPng.Int32(), int32(content.FileExt_FILE_EXT_PNG)},
		{"文件扩展名 gif", FileExtGif.Int32(), int32(content.FileExt_FILE_EXT_GIF)},
		{"文件扩展名 mp4", FileExtMp4.Int32(), int32(content.FileExt_FILE_EXT_MP4)},
		{"文件扩展名 mp3", FileExtMp3.Int32(), int32(content.FileExt_FILE_EXT_MP3)},
		{"文件扩展名 doc", FileExtDoc.Int32(), int32(content.FileExt_FILE_EXT_DOC)},
		{"文件扩展名 docx", FileExtDocx.Int32(), int32(content.FileExt_FILE_EXT_DOCX)},
		{"文件扩展名 pdf", FileExtPdf.Int32(), int32(content.FileExt_FILE_EXT_PDF)},
		{"文件扩展名 xls", FileExtXls.Int32(), int32(content.FileExt_FILE_EXT_XLS)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.proto, tt.biz)
		})
	}
}
