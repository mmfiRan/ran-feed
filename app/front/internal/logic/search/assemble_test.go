package search

import (
	"testing"

	"ran-feed/app/rpc/content/content"
	"ran-feed/app/rpc/search/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembleSearchContentItems(t *testing.T) {
	enriched := []*content.ContentItem{
		{ContentId: 2, Title: "标题2", AuthorName: "作者B", IsLiked: true, LikeCount: 5},
		nil,
		{ContentId: 1, Title: "标题1", AuthorName: "作者A"},
	}
	highlights := map[int64]*search.ContentHit{
		2: {ContentId: 2, HighlightTitle: "<em>标题</em>2", HighlightDescription: "简介命中"},
		// 1 无高亮
	}

	items := assembleSearchContentItems(enriched, highlights)
	require.Len(t, items, 2)

	// 顺序保持 content-rpc 返回顺序(即搜索排序) nil 被跳过
	assert.Equal(t, int64(2), items[0].ContentId)
	assert.Equal(t, int64(1), items[1].ContentId)

	// 命中项合并高亮 字段透传
	assert.Equal(t, "<em>标题</em>2", items[0].HighlightTitle)
	assert.Equal(t, "简介命中", items[0].HighlightDescription)
	assert.True(t, items[0].IsLiked)
	assert.Equal(t, int64(5), items[0].LikeCount)

	// 无高亮项留空
	assert.Empty(t, items[1].HighlightTitle)
	assert.Empty(t, items[1].HighlightDescription)
}
