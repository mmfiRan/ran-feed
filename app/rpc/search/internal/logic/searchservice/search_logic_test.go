package searchservicelogic

import (
	"testing"

	"ran-feed/app/rpc/search/internal/es"
	"ran-feed/app/rpc/search/search"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeSize(t *testing.T) {
	assert.Equal(t, 10, normalizeSize(0))   // 零值取默认
	assert.Equal(t, 10, normalizeSize(-1))  // 负值取默认
	assert.Equal(t, 20, normalizeSize(20))  // 正常透传
	assert.Equal(t, 50, normalizeSize(100)) // 超上限被截
}

func TestCursorRoundTrip(t *testing.T) {
	// content 游标混合类型:score float、hot_score float、published_at int、content_id string
	sortVals := []any{12.34, 8.5, float64(1_700_000_000_000), "12345"}
	cursor := encodeCursor(sortVals)
	require.NotEmpty(t, cursor)

	got, err := decodeCursor(cursor)
	require.NoError(t, err)
	assert.Equal(t, sortVals, got)
}

func TestCursorEmptyAndInvalid(t *testing.T) {
	assert.Empty(t, encodeCursor(nil)) // 空数组编码为空串

	got, err := decodeCursor("") // 空游标即首页
	require.NoError(t, err)
	assert.Nil(t, got)

	_, err = decodeCursor("!!!not-base64!!!") // 非法游标报错
	assert.Error(t, err)
}

func TestNextCursor(t *testing.T) {
	hits := []es.Hit{
		{ID: "1", Sort: []any{1.0, "1"}},
		{ID: "2", Sort: []any{2.0, "2"}},
	}
	// 满页 返回末条游标
	require.NotEmpty(t, nextCursor(hits, 2))
	decoded, err := decodeCursor(nextCursor(hits, 2))
	require.NoError(t, err)
	assert.Equal(t, []any{2.0, "2"}, decoded)

	// 不足一页 到底 返回空
	assert.Empty(t, nextCursor(hits, 10))
	// 空命中 返回空
	assert.Empty(t, nextCursor(nil, 10))
}

func TestBuildContentQuery(t *testing.T) {
	l := &SearchContentLogic{}

	t.Run("首页 无 from 无 search_after sort 末位 content_id 兜底", func(t *testing.T) {
		q := l.buildQuery(&search.SearchContentReq{Keyword: "露营"}, 20, nil)
		assert.NotContains(t, q, "from")
		assert.NotContains(t, q, "search_after")
		assert.Equal(t, 20, q["size"])

		boolq := q["query"].(map[string]any)["bool"].(map[string]any)
		mm := boolq["must"].([]map[string]any)[0]["multi_match"].(map[string]any)
		assert.Equal(t, "露营", mm["query"])
		assert.Equal(t, []string{"title^3", "description^2", "body"}, mm["fields"])
		assert.Equal(t, "ik_smart", mm["analyzer"])

		assert.Len(t, boolq["filter"].([]map[string]any), 3)
		sort := q["sort"].([]map[string]any)
		require.Len(t, sort, 4)
		assert.Contains(t, sort[3], "content_id")

		hlFields := q["highlight"].(map[string]any)["fields"].(map[string]any)
		assert.Contains(t, hlFields, "title")
		assert.Contains(t, hlFields, "description")
	})

	t.Run("带视频类型 多一个过滤", func(t *testing.T) {
		q := l.buildQuery(&search.SearchContentReq{Keyword: "露营", ContentType: search.ContentType_VIDEO}, 10, nil)
		filters := q["query"].(map[string]any)["bool"].(map[string]any)["filter"].([]map[string]any)
		require.Len(t, filters, 4)
		assert.Equal(t, int32(20), filters[3]["term"].(map[string]any)["content_type"])
	})

	t.Run("带游标 接 search_after", func(t *testing.T) {
		after := []any{5.0, 1.0, float64(123), "9"}
		q := l.buildQuery(&search.SearchContentReq{Keyword: "露营"}, 10, after)
		assert.Equal(t, after, q["search_after"])
	})
}

func TestBuildUserQuery(t *testing.T) {
	l := &SearchUserLogic{}

	q := l.buildQuery(&search.SearchUserReq{Keyword: "爱丽丝"}, 10, nil)
	assert.NotContains(t, q, "from")
	assert.NotContains(t, q, "search_after")
	boolq := q["query"].(map[string]any)["bool"].(map[string]any)
	mm := boolq["must"].([]map[string]any)[0]["multi_match"].(map[string]any)
	assert.Equal(t, "爱丽丝", mm["query"])
	assert.Equal(t, []string{"nickname^3", "bio"}, mm["fields"])
	assert.Len(t, boolq["filter"].([]map[string]any), 2)
	sort := q["sort"].([]map[string]any)
	require.Len(t, sort, 2)
	assert.Contains(t, sort[1], "user_id")

	q2 := l.buildQuery(&search.SearchUserReq{Keyword: "爱丽丝"}, 10, []any{3.0, "7"})
	assert.Equal(t, []any{3.0, "7"}, q2["search_after"])
}
