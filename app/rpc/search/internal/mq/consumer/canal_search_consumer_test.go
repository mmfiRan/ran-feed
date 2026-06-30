package consumer

import (
	"testing"

	"ran-feed/app/rpc/search/internal/common/consts"
	"ran-feed/app/rpc/search/internal/entity/model"

	"github.com/stretchr/testify/assert"
)

func TestClassifyContent(t *testing.T) {
	ids := []int64{1, 2, 3, 4}
	rows := map[int64]*model.RanFeedContent{
		1: {ID: 1, Status: consts.ContentStatusPublished, Visibility: consts.ContentVisibilityPublic}, // upsert
		2: {ID: 2, Status: 10, Visibility: consts.ContentVisibilityPublic},                            // 草稿 delete
		3: {ID: 3, Status: consts.ContentStatusPublished, Visibility: 20},                             // 私密 delete
		// 4 缺失(软删/硬删) delete
	}

	upserts, deleteIDs := classifyContent(ids, rows)
	assert.Len(t, upserts, 1)
	assert.Equal(t, int64(1), upserts[0].ID)
	assert.ElementsMatch(t, []int64{2, 3, 4}, deleteIDs)
}

func TestClassifyUser(t *testing.T) {
	ids := []int64{10, 20, 30}
	rows := map[int64]*model.RanFeedUser{
		10: {ID: 10, Status: consts.UserStatusNormal}, // upsert
		20: {ID: 20, Status: 20},                      // 封禁 delete
		// 30 缺失 delete
	}

	upserts, deleteIDs := classifyUser(ids, rows)
	assert.Len(t, upserts, 1)
	assert.Equal(t, int64(10), upserts[0].ID)
	assert.ElementsMatch(t, []int64{20, 30}, deleteIDs)
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		name string
		in   interface{}
		want int64
		ok   bool
	}{
		{name: "canal 字符串", in: "123", want: 123, ok: true},
		{name: "float64", in: float64(45), want: 45, ok: true},
		{name: "int64", in: int64(7), want: 7, ok: true},
		{name: "空串", in: "", want: 0, ok: false},
		{name: "nil", in: nil, want: 0, ok: false},
		{name: "非数字", in: "abc", want: 0, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseInt64(tt.in)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRowEventIDStable(t *testing.T) {
	row := map[string]interface{}{"id": "100"}
	a := rowEventID("evt", "ran_feed_content", "UPDATE", row, 0)
	b := rowEventID("evt", "ran_feed_content", "UPDATE", row, 0)
	assert.Equal(t, a, b)

	other := rowEventID("evt", "ran_feed_content", "UPDATE", map[string]interface{}{"id": "200"}, 0)
	assert.NotEqual(t, a, other)
}
