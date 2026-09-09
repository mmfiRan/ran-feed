package strategy

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"ran-feed/app/rpc/count/count"
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
type Registry struct {
	strategies map[string]TableStrategy
}

func newRegistry(strategies ...TableStrategy) *Registry {
	r := &Registry{strategies: make(map[string]TableStrategy, len(strategies))}
	for _, s := range strategies {
		r.register(s)
	}
	return r
}

func (r *Registry) register(s TableStrategy) {
	if s == nil {
		return
	}
	table := normalizeTableName(s.TableName())
	if table == "" {
		return
	}
	r.strategies[table] = s
}

func (r *Registry) Get(table string) (TableStrategy, bool) {
	s, ok := r.strategies[normalizeTableName(table)]
	return s, ok
}

func normalizeTableName(table string) string {
	return strings.ToLower(strings.TrimSpace(table))
}

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
	return newRegistry(strategies...)
}

// ParseInt64 把 canal 行字段统一解析为 int64 兼容数值字符串与 json.Number
func ParseInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case nil:
		return 0, false
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		return int64(n), true
	case float64:
		return int64(n), true
	case json.Number:
		val, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return val, true
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return 0, false
		}
		val, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, false
		}
		return val, true
	default:
		return 0, false
	}
}
