// Package convert content 域枚举的边界转换
package convert

import (
	"ran-feed/app/rpc/content/content"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/pkg/commonpb"
	"ran-feed/pkg/enums"
)

// ContentStatusFromPB 请求内容状态转业务枚举
func ContentStatusFromPB(v content.ContentStatus) (contentEnum.ContentStatusEnum, bool) {
	return enums.Parse[contentEnum.ContentStatusEnum](int32(v))
}

// VisibilityFromPB 请求可见性转业务枚举
// Parse 对 UNSPECIFIED 返回 ok 因为业务枚举的 0 值 Unknown 也在合法集合内 调用方须先单独判未指定
func VisibilityFromPB(v content.Visibility) (contentEnum.VisibilityEnum, bool) {
	return enums.Parse[contentEnum.VisibilityEnum](int32(v))
}

// ReviewDecisionFromPB 请求审核决策转业务枚举
func ReviewDecisionFromPB(v content.ReviewDecision) (contentEnum.ReviewDecisionEnum, bool) {
	return enums.Parse[contentEnum.ReviewDecisionEnum](int32(v))
}

// UploadSceneFromPB 请求上传场景转业务枚举
func UploadSceneFromPB(v content.UploadScene) (contentEnum.UploadSceneEnum, bool) {
	return enums.Parse[contentEnum.UploadSceneEnum](int32(v))
}

// FileExtFromPB 请求文件扩展名转业务枚举
func FileExtFromPB(v content.FileExt) (contentEnum.FileExtEnum, bool) {
	return enums.Parse[contentEnum.FileExtEnum](int32(v))
}

// ContentStatusPtrFromPB 可选状态筛选转业务枚举指针 未传返回 nil
func ContentStatusPtrFromPB(v *content.ContentStatus) (*contentEnum.ContentStatusEnum, bool) {
	if v == nil {
		return nil, true
	}
	s, ok := enums.Parse[contentEnum.ContentStatusEnum](int32(*v))
	if !ok {
		return nil, false
	}
	return &s, true
}

// ContentTypePtrFromPB 可选类型筛选转业务枚举指针 未传返回 nil
func ContentTypePtrFromPB(v *content.ContentType) (*contentEnum.ContentTypeEnum, bool) {
	if v == nil {
		return nil, true
	}
	t, ok := enums.Parse[contentEnum.ContentTypeEnum](int32(*v))
	if !ok {
		return nil, false
	}
	return &t, true
}

// ContentTypeValue 内容类型转统一枚举响应
func ContentTypeValue(contentType int32) *commonpb.EnumValue {
	return enums.ToCommonPB(contentEnum.ContentTypeEnum(contentType))
}

// ContentStatusValue 内容状态转统一枚举响应
func ContentStatusValue(status int32) *commonpb.EnumValue {
	return enums.ToCommonPB(contentEnum.ContentStatusEnum(status))
}

// VisibilityValue 可见性转统一枚举响应
func VisibilityValue(visibility int32) *commonpb.EnumValue {
	return enums.ToCommonPB(contentEnum.VisibilityEnum(visibility))
}
