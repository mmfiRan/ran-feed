package utils

// CastPtr nil 安全指针类型转换
func CastPtr[T Integer, U Integer](v *U) *T {
	if v == nil {
		return nil
	}
	t := T(*v)
	return &t
}
