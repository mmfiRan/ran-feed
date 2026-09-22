package strategy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 注意 本文件测 strategy 包内部 API 不 import presence 免 cycle
// 四表默认注册的验证放在 presence 包(其 init 已加载)
// 注册表容器语义由 pkg/event/registry 自测 此处只测域内装配

type fakeStrategy struct{ table string }

func (f *fakeStrategy) TableName() string { return f.table }
func (f *fakeStrategy) ExtractEvents(context.Context, string, map[string]interface{}, map[string]interface{}) []NotifyEvent {
	return nil
}

func TestNewDefaultRegistry_跳过Nil工厂(t *testing.T) {
	require.NotPanics(t, func() { NewDefaultRegistry() })
}

func TestRegisterFactory_Nil忽略(t *testing.T) {
	before := len(factories)
	RegisterFactory(nil)
	assert.Equal(t, before, len(factories), "nil 工厂不入表")
}
