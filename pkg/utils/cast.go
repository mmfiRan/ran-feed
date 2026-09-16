package utils

// CastPtr nil 安全指针类型转换
func CastPtr[T Integer, U Integer](v *U) *T {
	if v == nil {
		return nil
	}
	t := T(*v)
	return &t
}

// Deref 取指针值 nil 返回零值
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// PtrOrNil 空字符串转 nil
func PtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
