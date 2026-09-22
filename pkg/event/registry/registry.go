// Package registry canal 表策略注册表
package registry

import "strings"

type TableNamer interface {
	TableName() string
}

// Registry 表名到策略的映射
type Registry[S TableNamer] struct {
	strategies map[string]S
}

// New 建注册表 表名为空的策略被忽略
func New[S TableNamer](strategies ...S) *Registry[S] {
	r := &Registry[S]{
		strategies: make(map[string]S, len(strategies)),
	}
	for _, s := range strategies {
		table := NormalizeTable(s.TableName())
		if table == "" {
			continue
		}
		r.strategies[table] = s
	}
	return r
}

// Get 按表名取策略 未注册返回 false
func (r *Registry[S]) Get(table string) (S, bool) {
	s, ok := r.strategies[NormalizeTable(table)]
	return s, ok
}

func NormalizeTable(table string) string {
	return strings.ToLower(strings.TrimSpace(table))
}
