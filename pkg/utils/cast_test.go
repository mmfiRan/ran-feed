package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// castStatus 模拟 db 枚举具名类型 底层 int32
type castStatus int32

func TestCastPtr(t *testing.T) {
	t.Run("nil 原样返回", func(t *testing.T) {
		assert.Nil(t, CastPtr[int32]((*castStatus)(nil)))
	})
	t.Run("具名类型转基础类型", func(t *testing.T) {
		v := castStatus(20)
		got := CastPtr[int32](&v)
		assert.NotNil(t, got)
		assert.Equal(t, int32(20), *got)
	})
	t.Run("基础类型转具名类型", func(t *testing.T) {
		v := int32(10)
		got := CastPtr[castStatus](&v)
		assert.NotNil(t, got)
		assert.Equal(t, castStatus(10), *got)
	})
}
