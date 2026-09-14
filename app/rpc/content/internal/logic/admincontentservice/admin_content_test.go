package admincontentservicelogic

import (
	"testing"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/common/utils"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/count/count"

	"github.com/stretchr/testify/assert"
)

func TestFlipSourceStatus(t *testing.T) {
	tests := []struct {
		name    string
		target  content.ContentStatus
		want    content.ContentStatus
		wantErr bool
	}{
		{"下架 源自已发布", content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN, content.ContentStatus_CONTENT_STATUS_PUBLISHED, false},
		{"恢复 源自已下架", content.ContentStatus_CONTENT_STATUS_PUBLISHED, content.ContentStatus_CONTENT_STATUS_TAKEN_DOWN, false},
		{"草稿不支持", content.ContentStatus_CONTENT_STATUS_DRAFT, content.ContentStatus_CONTENT_STATUS_UNSPECIFIED, true},
		{"待审不支持", content.ContentStatus_CONTENT_STATUS_PENDING_REVIEW, content.ContentStatus_CONTENT_STATUS_UNSPECIFIED, true},
		{"拒绝不支持", content.ContentStatus_CONTENT_STATUS_REJECTED, content.ContentStatus_CONTENT_STATUS_UNSPECIFIED, true},
		{"未指定不支持", content.ContentStatus_CONTENT_STATUS_UNSPECIFIED, content.ContentStatus_CONTENT_STATUS_UNSPECIFIED, true},
	}
	l := &AdminSetContentStatusLogic{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := l.flipSourceStatus(tt.target)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildAdminContentItem(t *testing.T) {
	published := time.UnixMilli(1_700_000_000_000)
	created := time.UnixMilli(1_699_000_000_000)

	t.Run("已发布带发布时间", func(t *testing.T) {
		row := &model.RanFeedContent{
			ID:          10,
			UserID:      99,
			ContentType: int32(content.ContentType_CONTENT_TYPE_ARTICLE),
			Status:      int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED),
			Visibility:  int32(content.Visibility_VISIBILITY_PUBLIC),
			PublishedAt: &published,
			CreatedAt:   created,
		}
		counts := &count.ContentCountsItem{ContentId: 10, LikeCount: 3, FavoriteCount: 2, CommentCount: 1}
		item := buildAdminContentItem(row, "标题A", "user99", counts)
		assert.Equal(t, int64(10), item.ContentId)
		assert.Equal(t, "user99", item.Username)
		assert.Equal(t, utils.ContentTypeValue(int32(content.ContentType_CONTENT_TYPE_ARTICLE)), item.ContentType)
		assert.Equal(t, utils.ContentStatusValue(int32(content.ContentStatus_CONTENT_STATUS_PUBLISHED)), item.Status)
		assert.Equal(t, "标题A", item.Title)
		assert.Equal(t, int64(3), item.LikeCount)
		assert.Equal(t, int64(2), item.FavoriteCount)
		assert.Equal(t, int64(1), item.CommentCount)
		assert.Equal(t, published.UnixMilli(), item.PublishedAt.AsTime().UnixMilli())
		assert.Equal(t, created.UnixMilli(), item.CreatedAt.AsTime().UnixMilli())
	})

	t.Run("计数缺省归零", func(t *testing.T) {
		row := &model.RanFeedContent{
			ID:        12,
			CreatedAt: created,
		}
		item := buildAdminContentItem(row, "", "", nil)
		assert.Equal(t, int64(0), item.LikeCount)
		assert.Equal(t, int64(0), item.FavoriteCount)
		assert.Equal(t, int64(0), item.CommentCount)
	})

	t.Run("未发布 published_at 归零", func(t *testing.T) {
		row := &model.RanFeedContent{
			ID:          11,
			ContentType: int32(content.ContentType_CONTENT_TYPE_VIDEO),
			Status:      int32(content.ContentStatus_CONTENT_STATUS_DRAFT),
			CreatedAt:   created,
		}
		item := buildAdminContentItem(row, "", "", nil)
		assert.Equal(t, int64(0), item.PublishedAt.AsTime().UnixMilli())
		assert.Equal(t, utils.ContentTypeValue(int32(content.ContentType_CONTENT_TYPE_VIDEO)), item.ContentType)
	})
}
