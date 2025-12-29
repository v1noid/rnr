package utils

func Some[T any](args []T, fn func(T) bool) bool {
	for _, v := range args {
		if fn(v) {
			return true
		}
	}
	return false
}
