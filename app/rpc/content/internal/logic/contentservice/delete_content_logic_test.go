package contentservicelogic

import (
	"context"
	"testing"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/content/internal/repositories"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zeromicro/go-zero/core/logx"
)

// briefRepoMock 只覆盖 GetByIDBrief 其余嵌接口占位
type briefRepoMock struct {
	repositories.ContentRepository
	row *model.RanFeedContent
	err error
}

func (m *briefRepoMock) GetByIDBrief(_ context.Context, _ int64) (*model.RanFeedContent, error) {
	return m.row, m.err
}

func newTestDeleteLogic(repo repositories.ContentRepository) *DeleteContentLogic {
	return &DeleteContentLogic{
		ctx:         context.Background(),
		Logger:      logx.WithContext(context.Background()),
		contentRepo: repo,
	}
}

// TestDeleteContent_MissingRow 主表查不到 已软删 与非法 id 都返回 nil,nil
// 不判空就会在 row.UserID 处解引用崩 三种入参都要走到报错而不是 panic
func TestDeleteContent_MissingRow(t *testing.T) {
	found := &model.RanFeedContent{ID: 100, UserID: 7}

	tests := []struct {
		name    string
		repo    repositories.ContentRepository
		req     *content.DeleteContentReq
		wantMsg string
	}{
		{
			name:    "查不到返回 nil 报不存在",
			repo:    &briefRepoMock{row: nil},
			req:     &content.DeleteContentReq{ContentId: 999, UserId: 7},
			wantMsg: "内容不存在或无权限",
		},
		{
			name:    "非法 id 返回 nil 报不存在",
			repo:    &briefRepoMock{row: nil},
			req:     &content.DeleteContentReq{ContentId: 0, UserId: 7},
			wantMsg: "内容不存在或无权限",
		},
		{
			name:    "非本人内容报无权限",
			repo:    &briefRepoMock{row: found},
			req:     &content.DeleteContentReq{ContentId: 100, UserId: 8},
			wantMsg: "不是发布内容用户无法删除该内容",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newTestDeleteLogic(tt.repo).DeleteContent(tt.req)
			require.Error(t, err)
			assert.Equal(t, tt.wantMsg, err.Error())
		})
	}
}
