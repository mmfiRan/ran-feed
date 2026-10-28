package enums

import "fmt"

// UploadSceneEnum 上传场景
type UploadSceneEnum int32

const (
	UploadSceneUnknown      UploadSceneEnum = 0
	UploadSceneArticleCover UploadSceneEnum = 1
	UploadSceneVideo        UploadSceneEnum = 2
	UploadSceneAvatar       UploadSceneEnum = 3
	UploadSceneImImage      UploadSceneEnum = 4
	UploadSceneImVideo      UploadSceneEnum = 5
)

var uploadSceneNames = map[UploadSceneEnum]string{
	UploadSceneUnknown:      "UNKNOWN",
	UploadSceneArticleCover: "ARTICLE_COVER",
	UploadSceneVideo:        "VIDEO",
	UploadSceneAvatar:       "AVATAR",
	UploadSceneImImage:      "IM_IMAGE",
	UploadSceneImVideo:      "IM_VIDEO",
}

var uploadSceneMessages = map[UploadSceneEnum]string{
	UploadSceneUnknown:      "未知",
	UploadSceneArticleCover: "文章封面",
	UploadSceneVideo:        "视频",
	UploadSceneAvatar:       "头像",
	UploadSceneImImage:      "IM图片",
	UploadSceneImVideo:      "IM视频",
}

func (s UploadSceneEnum) Int32() int32 {
	return int32(s)
}

func (s UploadSceneEnum) Valid() bool {
	_, ok := uploadSceneNames[s]
	return ok
}

func (s UploadSceneEnum) String() string {
	if name, ok := uploadSceneNames[s]; ok {
		return name
	}
	return fmt.Sprintf("UploadSceneEnum(%d)", s)
}

func (s UploadSceneEnum) Message() string {
	if msg, ok := uploadSceneMessages[s]; ok {
		return msg
	}
	return fmt.Sprintf("未知(%d)", s)
}

// Path 场景对应对象键路径段
func (s UploadSceneEnum) Path() string {
	switch s {
	case UploadSceneArticleCover:
		return "article"
	case UploadSceneVideo:
		return "video"
	case UploadSceneAvatar:
		return "avatar"
	case UploadSceneImImage:
		return "im/image"
	case UploadSceneImVideo:
		return "im/video"
	default:
		return "unknown"
	}
}
