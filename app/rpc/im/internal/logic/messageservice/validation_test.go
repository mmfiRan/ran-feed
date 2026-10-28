package messageservicelogic

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"ran-feed/app/rpc/im/im"
	"ran-feed/app/rpc/im/internal/common/consts"
	imEnum "ran-feed/app/rpc/im/internal/common/enums"
)

func TestValidateSendMessage(t *testing.T) {
	base := func() *im.SendMessageReq {
		return &im.SendMessageReq{
			UserId:         1,
			ConversationId: 2,
			ClientMsgId:    "c-1",
			ContentType:    im.MessageContentType_MESSAGE_CONTENT_TYPE_TEXT,
			Content:        "hello",
		}
	}

	tests := []struct {
		name    string
		mutate  func(*im.SendMessageReq)
		content imEnum.ContentTypeEnum
		wantErr bool
	}{
		{name: "文本合法", content: imEnum.ContentTypeText},
		{name: "用户非法", mutate: func(r *im.SendMessageReq) { r.UserId = 0 }, content: imEnum.ContentTypeText, wantErr: true},
		{name: "会话非法", mutate: func(r *im.SendMessageReq) { r.ConversationId = 0 }, content: imEnum.ContentTypeText, wantErr: true},
		{name: "客户端标识为空", mutate: func(r *im.SendMessageReq) { r.ClientMsgId = "" }, content: imEnum.ContentTypeText, wantErr: true},
		{name: "客户端标识过长", mutate: func(r *im.SendMessageReq) { r.ClientMsgId = string(make([]byte, 65)) }, content: imEnum.ContentTypeText, wantErr: true},
		{name: "内容类型未知", content: imEnum.ContentTypeUnknown, wantErr: true},
		{name: "文本空白", mutate: func(r *im.SendMessageReq) { r.Content = "   " }, content: imEnum.ContentTypeText, wantErr: true},
		{name: "内容超长", mutate: func(r *im.SendMessageReq) { r.Content = strings.Repeat("a", consts.MaxContentLength+1) }, content: imEnum.ContentTypeText, wantErr: true},
		{
			name: "图片合法",
			mutate: func(r *im.SendMessageReq) {
				r.Content = `{"url":"https://oss.example.com/a.jpg","width":1,"height":1,"size":10}`
			},
			content: imEnum.ContentTypeImage,
		},
		{
			name:    "图片缺 url",
			mutate:  func(r *im.SendMessageReq) { r.Content = `{"width":1}` },
			content: imEnum.ContentTypeImage,
			wantErr: true,
		},
		{
			name:    "图片非 JSON",
			mutate:  func(r *im.SendMessageReq) { r.Content = "not-json" },
			content: imEnum.ContentTypeImage,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := base()
			if tc.mutate != nil {
				tc.mutate(req)
			}
			err := validateSendMessage(req, tc.content)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
