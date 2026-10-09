package admincontentservicelogic

import (
	"testing"
	"time"

	"ran-feed/app/rpc/content/internal/common/convert"
	contentEnum "ran-feed/app/rpc/content/internal/common/enums"
	"ran-feed/app/rpc/content/internal/entity/model"
	"ran-feed/app/rpc/count/count"

	"github.com/stretchr/testify/assert"
)

func TestFlipSourceStatus(t *testing.T) {
	tests := []struct {
		name    string
		target  contentEnum.ContentStatusEnum
		want    contentEnum.ContentStatusEnum
		wantErr bool
	}{
		{"下架 源自已发布", contentEnum.ContentStatusTakenDown, contentEnum.ContentStatusPublished, false},
		{"恢复 源自已下架", contentEnum.ContentStatusPublished, contentEnum.ContentStatusTakenDown, false},
		{"草稿不支持", contentEnum.ContentStatusDraft, contentEnum.ContentStatusUnknown, true},
		{"待审不支持", contentEnum.ContentStatusPendingReview, contentEnum.ContentStatusUnknown, true},
		{"拒绝不支持", contentEnum.ContentStatusRejected, contentEnum.ContentStatusUnknown, true},
		{"未指定不支持", contentEnum.ContentStatusUnknown, contentEnum.ContentStatusUnknown, true},
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
	l := &AdminListContentsLogic{}
	published := time.UnixMilli(1_700_000_000_000)
	created := time.UnixMilli(1_699_000_000_000)

	t.Run("已发布带发布时间", func(t *testing.T) {
		row := &model.RanFeedContent{
			ID:          10,
			UserID:      99,
			ContentType: contentEnum.ContentTypeArticle.Int32(),
			Status:      contentEnum.ContentStatusPublished.Int32(),
			Visibility:  contentEnum.VisibilityPublic.Int32(),
			PublishedAt: &published,
			CreatedAt:   created,
		}
		counts := &count.ContentCountsItem{ContentId: 10, LikeCount: 3, FavoriteCount: 2, CommentCount: 1}
		item := l.buildAdminContentItem(row, "标题A", "user99", counts)
		assert.Equal(t, int64(10), item.ContentId)
		assert.Equal(t, "user99", item.Username)
		assert.Equal(t, convert.ContentTypeValue(contentEnum.ContentTypeArticle.Int32()), item.ContentType)
		assert.Equal(t, convert.ContentStatusValue(contentEnum.ContentStatusPublished.Int32()), item.Status)
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
		item := l.buildAdminContentItem(row, "", "", nil)
		assert.Equal(t, int64(0), item.LikeCount)
		assert.Equal(t, int64(0), item.FavoriteCount)
		assert.Equal(t, int64(0), item.CommentCount)
	})

	t.Run("未发布 published_at 归零", func(t *testing.T) {
		row := &model.RanFeedContent{
			ID:          11,
			ContentType: contentEnum.ContentTypeVideo.Int32(),
			Status:      contentEnum.ContentStatusDraft.Int32(),
			CreatedAt:   created,
		}
		item := l.buildAdminContentItem(row, "", "", nil)
		assert.Equal(t, int64(0), item.PublishedAt.AsTime().UnixMilli())
		assert.Equal(t, convert.ContentTypeValue(contentEnum.ContentTypeVideo.Int32()), item.ContentType)
	})
}
