package strategy

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInt64(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  int64
		ok    bool
	}{
		{name: "int", input: 42, want: 42, ok: true},
		{name: "int64", input: int64(42), want: 42, ok: true},
		{name: "float64", input: float64(42), want: 42, ok: true},
		{name: "json.Number", input: json.Number("42"), want: 42, ok: true},
		{name: "数值字符串", input: "42", want: 42, ok: true},
		{name: "空字符串", input: "", want: 0, ok: false},
		{name: "非数字字符串", input: "abc", want: 0, ok: false},
		{name: "nil", input: nil, want: 0, ok: false},
		{name: "bool 不支持", input: true, want: 0, ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseInt64(tt.input)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
