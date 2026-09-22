package canal

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wantSHA1 按已落库的哈希输入格式算期望值 锁死格式而非硬编码不透明十六进制串
func wantSHA1(t *testing.T, format string, args ...any) string {
	t.Helper()
	h := sha1.Sum(fmt.Appendf(nil, format, args...))
	return hex.EncodeToString(h[:])
}

func TestParse(t *testing.T) {
	msg, err := Parse(`{"id":123,"table":" Ran_Feed_Like ","type":" update ","ts":1700000000,"data":[{"id":"1"}],"old":[{"id":"1"}]}`)
	require.NoError(t, err)

	assert.Equal(t, "ran_feed_like", msg.Table(), "表名应归一化为小写去空白")
	assert.Equal(t, "UPDATE", msg.Op(), "操作类型应归一化为大写去空白")
	assert.Len(t, msg.Data, 1)
	assert.Equal(t, map[string]any{"id": "1"}, msg.OldRow(0))

	_, err = Parse(`{bad json`)
	assert.Error(t, err, "非法 json 应报错")
}

func TestMessage_OldRow越界返回nil(t *testing.T) {
	msg := &Message{Old: []map[string]any{{"id": "1"}}}

	assert.Nil(t, msg.OldRow(-1))
	assert.Nil(t, msg.OldRow(1))
	assert.NotNil(t, msg.OldRow(0))
}

func TestMessage_UpdatedAt秒毫秒自适应(t *testing.T) {
	tests := []struct {
		name string
		ts   int64
		want time.Time
	}{
		{name: "秒级时间戳", ts: 1700000000, want: time.Unix(1700000000, 0)},
		{name: "毫秒级时间戳", ts: 1700000000000, want: time.UnixMilli(1700000000000)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, (&Message{Ts: tt.ts}).UpdatedAt())
		})
	}

	t.Run("缺失取当前时间", func(t *testing.T) {
		before := time.Now()
		got := (&Message{Ts: 0}).UpdatedAt()
		assert.False(t, got.Before(before), "ts 非正时应取当前时间")
	})
}

func TestMessage_EventID(t *testing.T) {
	raw := `{"id":null,"table":"ran_feed_like"}`

	t.Run("取报文 id", func(t *testing.T) {
		assert.Equal(t, "123", (&Message{ID: 123}).EventID(raw))
	})

	t.Run("id 缺失回退报文 sha1", func(t *testing.T) {
		assert.Equal(t, wantSHA1(t, "%s", raw), (&Message{ID: nil}).EventID(raw))
	})

	t.Run("超长截断到列宽", func(t *testing.T) {
		long := strings.Repeat("a", 100)
		got := (&Message{ID: long}).EventID(raw)
		assert.Len(t, got, eventIDMaxLen, "超长 id 应截断适配去重表列宽")
	})
}

// TestRowEventID_哈希输入格式 去重表已落库记录依赖该格式 变更等于去重失效
func TestRowEventID_哈希输入格式(t *testing.T) {
	t.Run("有行主键用主键", func(t *testing.T) {
		row := map[string]any{"id": "100", "user_id": "9"}
		got := RowEventID("evt", "ran_feed_like", "UPDATE", row, 3)

		assert.Equal(t, wantSHA1(t, "%s|%s|%s|%d", "evt", "ran_feed_like", "UPDATE", int64(100)), got,
			"有主键时格式须为 eventID|table|op|rowID 且不含行下标")
	})

	t.Run("无行主键回退行内容加下标", func(t *testing.T) {
		row := map[string]any{"user_id": "9"}
		rowJSON, err := json.Marshal(row)
		require.NoError(t, err)
		got := RowEventID("evt", "ran_feed_like", "INSERT", row, 3)

		assert.Equal(t, wantSHA1(t, "%s|%s|%s|%d|%s", "evt", "ran_feed_like", "INSERT", 3, string(rowJSON)), got,
			"无主键时格式须为 eventID|table|op|idx|rowJSON")
	})

	t.Run("eventID 为空用 unknown 占位", func(t *testing.T) {
		row := map[string]any{"id": "100"}
		got := RowEventID("", "ran_feed_like", "UPDATE", row, 0)

		assert.Equal(t, wantSHA1(t, "%s|%s|%s|%d", unknownEventID, "ran_feed_like", "UPDATE", int64(100)), got)
	})

	t.Run("主键非正回退行内容", func(t *testing.T) {
		row := map[string]any{"id": "0"}
		rowJSON, err := json.Marshal(row)
		require.NoError(t, err)
		got := RowEventID("evt", "ran_feed_like", "UPDATE", row, 1)

		assert.Equal(t, wantSHA1(t, "%s|%s|%s|%d|%s", "evt", "ran_feed_like", "UPDATE", 1, string(rowJSON)), got,
			"主键非正视为无主键")
	})
}

func TestRowEventID_同输入稳定异输入不同(t *testing.T) {
	row := map[string]any{"id": "100"}

	a := RowEventID("evt", "ran_feed_content", "UPDATE", row, 0)
	b := RowEventID("evt", "ran_feed_content", "UPDATE", row, 0)
	assert.Equal(t, a, b, "同输入须稳定")

	other := RowEventID("evt", "ran_feed_content", "UPDATE", map[string]any{"id": "200"}, 0)
	assert.NotEqual(t, a, other, "不同行须不同")
}

func TestParseInt64(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want int64
		ok   bool
	}{
		{name: "字符串", in: "42", want: 42, ok: true},
		{name: "带空白字符串", in: " 42 ", want: 42, ok: true},
		{name: "空字符串", in: "", want: 0, ok: false},
		{name: "非数字字符串", in: "abc", want: 0, ok: false},
		{name: "int", in: 42, want: 42, ok: true},
		{name: "int32", in: int32(42), want: 42, ok: true},
		{name: "int64", in: int64(42), want: 42, ok: true},
		{name: "uint", in: uint(42), want: 42, ok: true},
		{name: "uint32", in: uint32(42), want: 42, ok: true},
		{name: "uint64", in: uint64(42), want: 42, ok: true},
		{name: "float64", in: float64(42), want: 42, ok: true},
		{name: "json.Number", in: json.Number("42"), want: 42, ok: true},
		{name: "非法 json.Number", in: json.Number("x"), want: 0, ok: false},
		{name: "nil", in: nil, want: 0, ok: false},
		{name: "不支持类型", in: []int{1}, want: 0, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseInt64(tt.in)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseString(t *testing.T) {
	tests := []struct {
		name string
		in   any
		want string
	}{
		{name: "字符串去空白", in: "  hi  ", want: "hi"},
		{name: "nil 返回空串", in: nil, want: ""},
		{name: "数值转串", in: 42, want: "42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ParseString(tt.in))
		})
	}
}
