// Package enum 项目级通用枚举
// 所有枚举类型都应实现 Enum 接口 通过表驱动方式集中维护值与名称映射
// 业务专属判断方法（如 IsActive）保留在具体类型上 不下放到接口
package enum

import "fmt"

// Enum 所有项目枚举类型实现的通用契约
// 实现该接口的类型可被通用工具函数（Parse / MustValid 等）统一处理
type Enum interface {
	fmt.Stringer
	// Int32 返回底层数值 用于 DB 持久化与 RPC 传输
	Int32() int32
	// Valid 是否为合法定义的枚举值
	Valid() bool
}

// Int32Enum 约束所有底层类型为 int32 的枚举
// 用于泛型工具函数 兼顾类型安全与代码复用
type Int32Enum interface {
	Enum
	~int32
}

// Parse 把底层 int32 安全转换为目标枚举类型 非法值返回 (零值, false)
// 调用示例 status, ok := enum.Parse[enum.RecordStatus](10)
func Parse[T Int32Enum](v int32) (T, bool) {
	e := T(v)
	return e, e.Valid()
}

// MustParse 同 Parse 非法值直接 panic
// 适用于配置初始化等不应失败的场景
func MustParse[T Int32Enum](v int32) T {
	e, ok := Parse[T](v)
	if !ok {
		panic(fmt.Sprintf("invalid enum value %d for %T", v, e))
	}
	return e
}

// MustValid 断言枚举值合法 否则 panic
func MustValid(e Enum) {
	if !e.Valid() {
		panic(fmt.Sprintf("invalid enum value: %s", e))
	}
}
