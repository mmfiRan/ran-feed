package indexer

import (
	"testing"
	"time"

	"ran-feed/app/rpc/search/internal/entity/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockArticleRepo struct {
	data map[int64]*model.RanFeedArticle
}

func (m *mockArticleRepo) GetByContentIDs(ids []int64) (map[int64]*model.RanFeedArticle, error) {
	res := make(map[int64]*model.RanFeedArticle, len(ids))
	for _, id := range ids {
		if a, ok := m.data[id]; ok {
			res[id] = a
		}
	}
	return res, nil
}

type mockVideoRepo struct {
	data map[int64]*model.RanFeedVideo
}

func (m *mockVideoRepo) GetByContentIDs(ids []int64) (map[int64]*model.RanFeedVideo, error) {
	res := make(map[int64]*model.RanFeedVideo, len(ids))
	for _, id := range ids {
		if v, ok := m.data[id]; ok {
			res[id] = v
		}
	}
	return res, nil
}

func strptr(s string) *string { return &s }

func TestAssembleContentDocs(t *testing.T) {
	publishedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updatedAt := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)

	assembler := NewAssembler(
		&mockArticleRepo{data: map[int64]*model.RanFeedArticle{
			1: {ContentID: 1, Title: "标题A", Description: strptr("简介A"), Content: "正文A"},
			3: {ContentID: 3, Title: "标题C", Description: nil, Content: "正文C"},
		}},
		&mockVideoRepo{data: map[int64]*model.RanFeedVideo{
			2: {ContentID: 2, Title: "视频B"},
		}},
	)

	contents := []*model.RanFeedContent{
		{ID: 1, UserID: 100, ContentType: 10, Status: 30, Visibility: 10, HotScore: 1.5, PublishedAt: &publishedAt, UpdatedAt: updatedAt},
		{ID: 2, UserID: 200, ContentType: 20, Status: 30, Visibility: 10, PublishedAt: nil, UpdatedAt: updatedAt},
		nil,
		{ID: 3, UserID: 300, ContentType: 10, Status: 30, Visibility: 10, PublishedAt: &publishedAt, UpdatedAt: updatedAt},
	}

	items, err := assembler.AssembleContentDocs(contents)
	require.NoError(t, err)
	require.Len(t, items, 3)

	// 文章 标题简介正文齐全 id 转字符串 version 取 updated_at 毫秒
	art := items[0]
	assert.Equal(t, "1", art.ID)
	assert.Equal(t, updatedAt.UnixMilli(), art.Version)
	artDoc := art.Doc.(ContentDoc)
	assert.Equal(t, "1", artDoc.ContentID)
	assert.Equal(t, "100", artDoc.AuthorID)
	assert.Equal(t, "标题A", artDoc.Title)
	assert.Equal(t, "简介A", artDoc.Description)
	assert.Equal(t, "正文A", artDoc.Body)
	assert.Equal(t, publishedAt.UnixMilli(), artDoc.PublishedAt)
	assert.InDelta(t, 1.5, artDoc.HotScore, 1e-9)

	// 视频 只有标题 无简介正文 published_at 为空转 0
	vidDoc := items[1].Doc.(ContentDoc)
	assert.Equal(t, "视频B", vidDoc.Title)
	assert.Empty(t, vidDoc.Description)
	assert.Empty(t, vidDoc.Body)
	assert.Zero(t, vidDoc.PublishedAt)

	// 文章 description 为 nil 指针转空串不 panic
	cDoc := items[2].Doc.(ContentDoc)
	assert.Equal(t, "标题C", cDoc.Title)
	assert.Empty(t, cDoc.Description)
}

func TestAssembleContentDocsEmpty(t *testing.T) {
	assembler := NewAssembler(&mockArticleRepo{}, &mockVideoRepo{})
	items, err := assembler.AssembleContentDocs(nil)
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestAssembleUserDocs(t *testing.T) {
	updatedAt := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	users := []*model.RanFeedUser{
		{ID: 10, Username: "alice", Nickname: "爱丽丝", Bio: "简介", Status: 10, UpdatedAt: updatedAt},
		nil,
	}

	items := AssembleUserDocs(users)
	require.Len(t, items, 1)
	assert.Equal(t, "10", items[0].ID)
	assert.Equal(t, updatedAt.UnixMilli(), items[0].Version)
	doc := items[0].Doc.(UserDoc)
	assert.Equal(t, "10", doc.UserID)
	assert.Equal(t, "alice", doc.Username)
	assert.Equal(t, "爱丽丝", doc.Nickname)
	assert.Equal(t, "简介", doc.Bio)
	assert.Equal(t, int32(10), doc.Status)
}
