package canal

import "fmt"

// OnlyIgnoredColumnsChanged 判断 UPDATE 是否只改动了 ignored 里的列
// 需要 old 行可比对 缺失或不完整时保守返回 false 保证不漏处理
// 变更集合非空且全部落在 ignored 内才返回 true
func OnlyIgnoredColumnsChanged(row, oldRow map[string]any, ignored ...string) bool {
	if len(row) == 0 || len(oldRow) == 0 {
		return false
	}
	ignoredSet := make(map[string]struct{}, len(ignored))
	for _, col := range ignored {
		ignoredSet[col] = struct{}{}
	}

	changed := false
	for col, newVal := range row {
		oldVal, ok := oldRow[col]
		if !ok {
			// old 缺该列 无法比对 保守按需处理
			return false
		}
		if sameColumnValue(newVal, oldVal) {
			continue
		}
		if _, skip := ignoredSet[col]; !skip {
			return false
		}
		changed = true
	}
	return changed
}

func sameColumnValue(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}
