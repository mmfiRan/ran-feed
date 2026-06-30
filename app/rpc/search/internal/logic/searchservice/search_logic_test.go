package searchservicelogic

import (
	"testing"

	"ran-feed/app/rpc/search/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageToFromSize(t *testing.T) {
	tests := []struct {
		name     string
		page     int32
		size     int32
		wantFrom int
		wantSize int
	}{
		{name: "零值取默认", page: 0, size: 0, wantFrom: 0, wantSize: 10},
		{name: "首页", page: 1, size: 20, wantFrom: 0, wantSize: 20},
		{name: "第三页", page: 3, size: 20, wantFrom: 40, wantSize: 20},
		{name: "size 超上限被截", page: 2, size: 100, wantFrom: 50, wantSize: 50},
		{name: "page 小于 1 归一", page: -1, size: 10, wantFrom: 0, wantSize: 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, size := pageToFromSize(tt.page, tt.size)
			assert.Equal(t, tt.wantFrom, from)
			assert.Equal(t, tt.wantSize, size)
		})
	}
}

func TestBuildContentQuery(t *testing.T) {
	l := &SearchContentLogic{}

	t.Run("不带类型 三个过滤", func(t *testing.T) {
		q := l.buildQuery(&search.SearchContentReq{Keyword: "露营"}, 10, 20)
		assert.Equal(t, 10, q["from"])
		assert.Equal(t, 20, q["size"])

		boolq := q["query"].(map[string]any)["bool"].(map[string]any)
		mm := boolq["must"].([]map[string]any)[0]["multi_match"].(map[string]any)
		assert.Equal(t, "露营", mm["query"])
		assert.Equal(t, []string{"title^3", "description^2", "body"}, mm["fields"])
		assert.Equal(t, "ik_smart", mm["analyzer"])

		assert.Len(t, boolq["filter"].([]map[string]any), 3)
		assert.Len(t, q["sort"].([]map[string]any), 3)

		hlFields := q["highlight"].(map[string]any)["fields"].(map[string]any)
		assert.Contains(t, hlFields, "title")
		assert.Contains(t, hlFields, "description")
	})

	t.Run("带视频类型 多一个过滤", func(t *testing.T) {
		q := l.buildQuery(&search.SearchContentReq{Keyword: "露营", ContentType: search.ContentType_VIDEO}, 0, 10)
		filters := q["query"].(map[string]any)["bool"].(map[string]any)["filter"].([]map[string]any)
		require.Len(t, filters, 4)
		assert.Equal(t, int32(20), filters[3]["term"].(map[string]any)["content_type"])
	})
}

func TestBuildUserQuery(t *testing.T) {
	l := &SearchUserLogic{}
	q := l.buildQuery(&search.SearchUserReq{Keyword: "爱丽丝"}, 0, 10)

	boolq := q["query"].(map[string]any)["bool"].(map[string]any)
	mm := boolq["must"].([]map[string]any)[0]["multi_match"].(map[string]any)
	assert.Equal(t, "爱丽丝", mm["query"])
	assert.Equal(t, []string{"nickname^3", "bio"}, mm["fields"])
	assert.Len(t, boolq["filter"].([]map[string]any), 2)
	assert.Len(t, q["sort"].([]map[string]any), 1)
}
