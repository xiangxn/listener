package tools

// Equaler 是一个接口，要求实现 Equal 方法
type Equaler[T any] interface {
	Equal(T) bool
}

// 去重函数，适用于任何实现 Equaler 接口的类型
func Unique[T Equaler[T]](input []T) []T {
	unique := make([]T, 0, len(input))
	for _, v := range input {
		found := false
		for _, u := range unique {
			if v.Equal(u) {
				found = true
				break
			}
		}
		if !found {
			unique = append(unique, v)
		}
	}
	return unique
}
