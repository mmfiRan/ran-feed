package admincontentservicelogic

import (
	"testing"
	"time"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/content/internal/entity/model"

	"github.com/stretchr/testify/assert"
)

func TestValidateStatusTransition(t *testing.T) {
	tests := []struct {
		name     string
		cur      content.ContentStatus
		target   content.ContentStatus
		wantNoop bool
		wantErr  bool
	}{
		{"下架 已发布->下架", content.ContentStatus_PUBLISHED, content.ContentStatus_TAKEN_DOWN, false, false},
		{"恢复 下架->已发布", content.ContentStatus_TAKEN_DOWN, content.ContentStatus_PUBLISHED, false, false},
		{"下架 幂等 已下架->下架", content.ContentStatus_TAKEN_DOWN, content.ContentStatus_TAKEN_DOWN, true, false},
		{"恢复 幂等 已发布->已发布", content.ContentStatus_PUBLISHED, content.ContentStatus_PUBLISHED, true, false},
		{"下架 草稿不可下架", content.ContentStatus_DRAFT, content.ContentStatus_TAKEN_DOWN, false, true},
		{"下架 待审不可下架", content.ContentStatus_PENDING_REVIEW, content.ContentStatus_TAKEN_DOWN, false, true},
		{"恢复 已发布不可恢复", content.ContentStatus_PUBLISHED, content.ContentStatus_TAKEN_DOWN, false, false}, // 属下架路径 见上
		{"恢复 草稿不可恢复", content.ContentStatus_DRAFT, content.ContentStatus_PUBLISHED, false, true},
		{"不支持的目标态 转草稿", content.ContentStatus_PUBLISHED, content.ContentStatus_DRAFT, false, true},
		{"不支持的目标态 转拒绝", content.ContentStatus_PUBLISHED, content.ContentStatus_REJECTED, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			noop, err := validateStatusTransition(tt.cur, tt.target)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantNoop, noop)
		})
	}
}

func TestBuildAdminContentItem(t *testing.T) {
	published := time.UnixMilli(1_700_000_000_000)
	created := time.UnixMilli(1_699_000_000_000)

	t.Run("已发布带发布时间", func(t *testing.T) {
		row := &model.RanFeedContent{
			ID:            10,
			UserID:        99,
			ContentType:   int32(content.ContentType_ARTICLE),
			Status:        int32(content.ContentStatus_PUBLISHED),
			Visibility:    int32(content.Visibility_PUBLIC),
			LikeCount:     3,
			FavoriteCount: 2,
			CommentCount:  1,
			PublishedAt:   &published,
			CreatedAt:     created,
		}
		item := buildAdminContentItem(row, "标题A")
		assert.Equal(t, int64(10), item.ContentId)
		assert.Equal(t, content.ContentType_ARTICLE, item.ContentType)
		assert.Equal(t, content.ContentStatus_PUBLISHED, item.Status)
		assert.Equal(t, "标题A", item.Title)
		assert.Equal(t, int64(3), item.LikeCount)
		assert.Equal(t, published.UnixMilli(), item.PublishedAt)
		assert.Equal(t, created.UnixMilli(), item.CreatedAt)
	})

	t.Run("未发布 published_at 归零", func(t *testing.T) {
		row := &model.RanFeedContent{
			ID:          11,
			ContentType: int32(content.ContentType_VIDEO),
			Status:      int32(content.ContentStatus_DRAFT),
			CreatedAt:   created,
		}
		item := buildAdminContentItem(row, "")
		assert.Equal(t, int64(0), item.PublishedAt)
		assert.Equal(t, content.ContentType_VIDEO, item.ContentType)
	})
}

func TestOptionalFilters(t *testing.T) {
	t.Run("零值转 nil", func(t *testing.T) {
		req := &content.AdminListContentsReq{}
		assert.Nil(t, optionalStatus(req))
		assert.Nil(t, optionalContentType(req))
		assert.Nil(t, optionalAuthorID(req))
	})
	t.Run("非零透传", func(t *testing.T) {
		s := content.ContentStatus_TAKEN_DOWN
		ct := content.ContentType_VIDEO
		req := &content.AdminListContentsReq{Status: &s, ContentType: &ct, AuthorId: ptrInt64(7)}
		assert.Equal(t, int32(content.ContentStatus_TAKEN_DOWN), *optionalStatus(req))
		assert.Equal(t, int32(content.ContentType_VIDEO), *optionalContentType(req))
		assert.Equal(t, int64(7), *optionalAuthorID(req))
	})
}

func ptrInt64(v int64) *int64 { return &v }

func TestBuildReviewDO(t *testing.T) {
	t.Run("拒绝带理由", func(t *testing.T) {
		d := buildReviewDO(88, reviewDecisionReject, "含敏感内容", 7)
		assert.Equal(t, int64(88), d.ContentID)
		assert.Equal(t, reviewDecisionReject, d.Decision)
		assert.Equal(t, "含敏感内容", d.Reason)
		assert.Equal(t, int64(7), d.CreatedBy)
		assert.Equal(t, int64(7), d.UpdatedBy)
		assert.NotZero(t, d.ID)
	})
	t.Run("通过理由为空", func(t *testing.T) {
		d := buildReviewDO(88, reviewDecisionApprove, "", 7)
		assert.Equal(t, reviewDecisionApprove, d.Decision)
		assert.Empty(t, d.Reason)
	})
}