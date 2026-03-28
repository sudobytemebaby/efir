package mapper

func Slice[T, U any](items []T, fn func(T) U) []U {
	result := make([]U, len(items))
	for i, item := range items {
		result[i] = fn(item)
	}
	return result
}

func Enum[From comparable, To any](m map[From]To, from From, fallback To) To {
	to, ok := m[from]
	if !ok {
		return fallback
	}
	return to
}

func EnumWithOk[From comparable, To any](m map[From]To, from From) (To, bool) {
	to, ok := m[from]
	return to, ok
}
