package messageservicelogic

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/consts"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
	"ran-feed/pkg/errorx"
)

// mediaContent 图片或视频消息 content 字段的 JSON 结构
type mediaContent struct {
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Size     int64  `json:"size"`
	Duration int64  `json:"duration"`
}

// validateSendMessage 校验发送参数,富媒体要求合法 JSON 且 url 非空
func validateSendMessage(in *im.SendMessageReq, contentType imEnum.ContentTypeEnum) error {
	if in.UserId <= 0 || in.ConversationId <= 0 {
		return errorx.NewMsg("参数错误")
	}
	if in.ClientMsgId == "" || utf8.RuneCountInString(in.ClientMsgId) > consts.MaxClientMsgIDLen {
		return errorx.NewMsg("客户端消息标识非法")
	}
	if contentType == imEnum.ContentTypeUnknown || !contentType.Valid() {
		return errorx.NewMsg("消息内容类型非法")
	}
	if utf8.RuneCountInString(in.Content) > consts.MaxContentLength {
		return errorx.NewMsg("消息内容过长")
	}

	switch contentType {
	case imEnum.ContentTypeText:
		if strings.TrimSpace(in.Content) == "" {
			return errorx.NewMsg("消息内容不能为空")
		}
	case imEnum.ContentTypeImage, imEnum.ContentTypeVideo:
		var media mediaContent
		if err := json.Unmarshal([]byte(in.Content), &media); err != nil || media.URL == "" {
			return errorx.NewMsg("富媒体消息格式非法")
		}
	}
	return nil
}
