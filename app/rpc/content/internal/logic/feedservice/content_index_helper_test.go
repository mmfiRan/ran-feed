package feedservicelogic

import (
	"testing"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/model"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestBuildContentIndexItem(t *testing.T) {
	publishedAt := time.UnixMilli(1_700_000_000_000)
	updatedAt := time.UnixMilli(1_700_000_123_456)
	desc := "文章摘要"

	tests := []struct {
		name    string
		content *model.RanFeedContent
		article *model.RanFeedArticle
		video   *model.RanFeedVideo
		expect  *content.ContentIndexItem
	}{
		{
			name: "文章_取标题简介正文",
			content: &model.RanFeedContent{
				ID: 1, UserID: 100,
				ContentType: int32(content.ContentType_CONTENT_TYPE_ARTICLE),
				Status:      int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED),
				Visibility:  int32(content.Visibility_VISIBILITY_PUBLIC),
				HotScore:    8.5,
				PublishedAt: &publishedAt,
				UpdatedAt:   updatedAt,
			},
			article: &model.RanFeedArticle{ContentID: 1, Title: "标题", Description: &desc, Content: "正文内容"},
			expect: &content.ContentIndexItem{
				ContentId: 1, ContentType: content.ContentType_CONTENT_TYPE_ARTICLE,
				Status: content.ContentStatus_CONTENT_STATUS_PUBLISHED, Visibility: content.Visibility_VISIBILITY_PUBLIC,
				AuthorId: 100, Title: "标题", Description: "文章摘要", Body: "正文内容",
				PublishedAt: timestamppb.New(time.UnixMilli(1_700_000_000_000)), HotScore: 8.5, Version: 1_700_000_123_456,
			},
		},
		{
			name: "视频_只有标题",
			content: &model.RanFeedContent{
				ID: 2, UserID: 200,
				ContentType: int32(content.ContentType_CONTENT_TYPE_VIDEO),
				Status:      int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED),
				Visibility:  int32(content.Visibility_VISIBILITY_PUBLIC),
				PublishedAt: &publishedAt,
				UpdatedAt:   updatedAt,
			},
			video: &model.RanFeedVideo{ContentID: 2, Title: "视频标题"},
			expect: &content.ContentIndexItem{
				ContentId: 2, ContentType: content.ContentType_CONTENT_TYPE_VIDEO,
				Status: content.ContentStatus_CONTENT_STATUS_PUBLISHED, Visibility: content.Visibility_VISIBILITY_PUBLIC,
				AuthorId: 200, Title: "视频标题",
				PublishedAt: timestamppb.New(time.UnixMilli(1_700_000_000_000)), Version: 1_700_000_123_456,
			},
		},
		{
			name: "文章无描述_子表缺失字段留空",
			content: &model.RanFeedContent{
				ID: 3, UserID: 300,
				ContentType: int32(content.ContentType_CONTENT_TYPE_ARTICLE),
				UpdatedAt:   updatedAt,
			},
			article: &model.RanFeedArticle{ContentID: 3, Title: "无描述", Description: nil, Content: "body"},
			expect: &content.ContentIndexItem{
				ContentId: 3, ContentType: content.ContentType_CONTENT_TYPE_ARTICLE, AuthorId: 300,
				Title: "无描述", Body: "body", PublishedAt: timestamppb.New(time.UnixMilli(0)), Version: 1_700_000_123_456,
			},
		},
		{
			name: "发布时间为空_published_at 归零",
			content: &model.RanFeedContent{
				ID: 4, ContentType: int32(content.ContentType_CONTENT_TYPE_ARTICLE), PublishedAt: nil, UpdatedAt: updatedAt,
			},
			article: &model.RanFeedArticle{ContentID: 4, Title: "t", Content: "b"},
			expect: &content.ContentIndexItem{
				ContentId: 4, ContentType: content.ContentType_CONTENT_TYPE_ARTICLE,
				Title: "t", Body: "b", PublishedAt: timestamppb.New(time.UnixMilli(0)), Version: 1_700_000_123_456,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildContentIndexItem(tt.content, tt.article, tt.video)
			assert.Equal(t, tt.expect.ContentId, got.ContentId)
			assert.Equal(t, tt.expect.ContentType, got.ContentType)
			assert.Equal(t, tt.expect.Status, got.Status)
			assert.Equal(t, tt.expect.Visibility, got.Visibility)
			assert.Equal(t, tt.expect.AuthorId, got.AuthorId)
			assert.Equal(t, tt.expect.Title, got.Title)
			assert.Equal(t, tt.expect.Description, got.Description)
			assert.Equal(t, tt.expect.Body, got.Body)
			assert.Equal(t, tt.expect.PublishedAt, got.PublishedAt)
			assert.Equal(t, tt.expect.HotScore, got.HotScore)
			assert.Equal(t, tt.expect.Version, got.Version)
		})
	}
}
