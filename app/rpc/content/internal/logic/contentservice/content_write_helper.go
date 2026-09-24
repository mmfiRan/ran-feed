package contentservicelogic

import (
	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/do"
	"ran-feed/pkg/errorx"
	"ran-feed/pkg/snowflake"
)

// contentWriteMode 写库模式 草稿放宽校验 发布与提交强校验
type contentWriteMode int

const (
	writeModeDraft contentWriteMode = iota
	writeModePublish
)

// buildContentDO 统一构造内容主行 四个建内容入口共用
func buildContentDO(userID int64, contentType content.ContentType, status content.ContentStatus, visibility int32) *do.ContentDO {
	return &do.ContentDO{
		ID:          snowflake.GenID(),
		UserID:      userID,
		ContentType: int32(contentType),
		Status:      int32(status),
		Visibility:  visibility,
		CreatedBy:   userID,
		UpdatedBy:   userID,
	}
}

// resolveWriteVisibility 解析可见性
// 草稿缺省回退 fallback(新建公开 编辑原值) 发布必须显式给合法值 防写入不属于取值域的 0
func resolveWriteVisibility(mode contentWriteMode, reqVis content.Visibility, fallback int32) (int32, error) {
	if reqVis == content.Visibility_VISIBILITY_UNSPECIFIED {
		if mode == writeModePublish {
			return 0, errorx.NewMsg("可见性不能为空")
		}
		return fallback, nil
	}
	if reqVis != content.Visibility_VISIBILITY_PUBLIC && reqVis != content.Visibility_VISIBILITY_PRIVATE {
		return 0, errorx.NewMsg("可见性取值非法")
	}
	return int32(reqVis), nil
}

// validateArticlePublish 发布文章完整性 RPC 层兜底 防直连 RPC 绕过前端静态校验
func validateArticlePublish(title, cover, body string) error {
	if title == "" || cover == "" || body == "" {
		return errorx.NewMsg("标题 封面 正文不能为空")
	}
	return nil
}

// validateVideoPublish 发布视频完整性 RPC 层兜底
func validateVideoPublish(title, cover, videoURL string) error {
	if title == "" || cover == "" || videoURL == "" {
		return errorx.NewMsg("标题 封面 视频不能为空")
	}
	return nil
}
