// Package enums 项目级枚举基础设施与通用枚举
// 业务域枚举实现 Enum 接口 各域在 internal/common/enums 自行维护
// 通用枚举（如软删除标记）在本包维护
package enums

import "fmt"

// Int 枚举底层数值约束 可比较整数类型
type Int interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Enum 业务枚举统一契约 各域自行实现
// 方法接口不含底层类型约束 可作为值类型使用
type Enum interface {
	fmt.Stringer
	// Int32 返回底层 int32 数值
	Int32() int32
	// Valid 是否为合法定义的枚举值
	Valid() bool
	// Message 中文描述 供前端展示
	Message() string
}

// IntEnum 组合底层整数约束与枚举契约 用于泛型工具函数
type IntEnum interface {
	Enum
	Int
}

// EnumValue 统一对外枚举结构 响应字段用 前端直接消费
type EnumValue struct {
	Code    int32  `json:"code"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// Value 把业务枚举转成统一响应结构
func Value[T IntEnum](e T) EnumValue {
	return EnumValue{
		Code:    e.Int32(),
		Name:    e.String(),
		Message: e.Message(),
	}
}

// Parse 把底层 int32 安全转换为目标枚举 非法返回 (零值, false)
func Parse[T IntEnum](v int32) (T, bool) {
	e := T(v)
	return e, e.Valid()
}

// MustParse 同 Parse 非法值直接 panic 适用于配置初始化等不应失败的场景
func MustParse[T IntEnum](v int32) T {
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
