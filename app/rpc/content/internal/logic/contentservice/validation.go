package contentservicelogic

import (
	"ran-feed/app/rpc/content/content"
	"ran-feed/pkg/errorx"
)

// contentWriteMode 区分草稿和发布两条写入路径的校验强度
type contentWriteMode int

const (
	writeModeDraft contentWriteMode = iota
	writeModePublish
)

// resolveWriteVisibility 定可见性 草稿缺省回退 fallback 新建默认公开编辑沿用原值 发布必须显式给合法值 否则请求里的 0 会被当合法枚举写进库
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

// validateArticlePublish 发布前完整性校验 放 RPC 层因为前端校验能被直连 RPC 绕过
func validateArticlePublish(title, cover, body string) error {
	if title == "" || cover == "" || body == "" {
		return errorx.NewMsg("标题 封面 正文不能为空")
	}
	return nil
}

// validateVideoPublish 发布前完整性校验 视频版
func validateVideoPublish(title, cover, videoURL string) error {
	if title == "" || cover == "" || videoURL == "" {
		return errorx.NewMsg("标题 封面 视频不能为空")
	}
	return nil
}
