package strategy

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/logc"

	"ran-feed/app/rpc/count/count"
)

// Update 表示一条计数增量更新。
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

// TableStrategy 定义某张表在 Canal 消息中的消费策略。
type TableStrategy interface {
	TableName() string
	ExtractUpdates(ctx context.Context, op string, row map[string]interface{}, oldRow map[string]interface{}) []Update
}

// Registry 管理 table -> strategy 的映射。
type Registry struct {
	strategies map[string]TableStrategy
}

func newRegistry(strategies ...TableStrategy) *Registry {
	r := &Registry{strategies: make(map[string]TableStrategy, len(strategies))}
	for _, s := range strategies {
		r.Register(s)
	}
	return r
}

func (r *Registry) Register(s TableStrategy) {
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

func registerFactory(factory func() TableStrategy) {
	if factory == nil {
		return
	}
	factories = append(factories, factory)
}

// NewDefaultRegistry 创建默认策略集合。
func NewDefaultRegistry() *Registry {
	strategies := make([]TableStrategy, 0, len(factories))
	for _, f := range factories {
		if f == nil {
			continue
		}
		s := f()
		if s == nil {
			continue
		}
		strategies = append(strategies, s)
	}
	return newRegistry(strategies...)
}

type contentCountTableStrategy struct {
	tableName   string
	bizType     count.BizType
	deltaByOpFn map[string]func(row map[string]interface{}, oldRow map[string]interface{}) int64
}

func (s *contentCountTableStrategy) TableName() string {
	return s.tableName
}

func (s *contentCountTableStrategy) ExtractUpdates(ctx context.Context, op string, row map[string]interface{}, oldRow map[string]interface{}) []Update {
	handler, ok := s.deltaByOpFn[strings.ToUpper(strings.TrimSpace(op))]
	if !ok || handler == nil {
		return nil
	}

	contentID, ok := getInt64(row["content_id"])
	if !ok || contentID <= 0 {
		logc.Errorf(ctx, "canal消息缺少有效content_id: table=%s, op=%s, row=%v", s.tableName, op, row)
		return nil
	}

	delta := handler(row, oldRow)
	if delta == 0 {
		return nil
	}

	ownerID, ok := getInt64(row["content_user_id"])
	if !ok || ownerID <= 0 {
		logc.Errorf(ctx, "canal消息缺少有效content_user_id: table=%s, op=%s, row=%v", s.tableName, op, row)
		ownerID = 0
	}

	return []Update{{
		BizType:    s.bizType,
		TargetType: count.TargetType_CONTENT,
		TargetID:   contentID,
		Delta:      delta,
		OwnerID:    ownerID,
	}}
}
