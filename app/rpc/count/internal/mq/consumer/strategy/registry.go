package strategy

import (
	"context"

	"ran-feed/app/rpc/count/count"
	"ran-feed/pkg/event/registry"
)

// Update 表示一条计数增量更新
type Update struct {
	BizType    count.BizType
	TargetType count.TargetType
	TargetID   int64
	Delta      int64
	OwnerID    int64
	Action     UpdateAction
}

type UpdateAction int

const (
	UpdateActionDelta UpdateAction = iota
	UpdateActionResetToZero
)

// TableStrategy 定义某张表在 Canal 消息中的消费策略
type TableStrategy interface {
	TableName() string
	ExtractUpdates(ctx context.Context, op string, row map[string]interface{}, oldRow map[string]interface{}) []Update
}

// Registry 管理 table 到 strategy 的映射
type Registry = registry.Registry[TableStrategy]

var factories []func() TableStrategy

// RegisterFactory 子包通过 init 注册各自的表策略工厂
func RegisterFactory(factory func() TableStrategy) {
	if factory == nil {
		return
	}
	factories = append(factories, factory)
}

// NewDefaultRegistry 实例化所有已注册的表策略
func NewDefaultRegistry() *Registry {
	strategies := make([]TableStrategy, 0, len(factories))
	for _, f := range factories {
		if f == nil {
			continue
		}
		if s := f(); s != nil {
			strategies = append(strategies, s)
		}
	}
	return registry.New(strategies...)
}
