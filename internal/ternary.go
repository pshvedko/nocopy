package internal

func Ternary[T any](b bool, t, f T) T {
	if b {
		return t
	}
	return f
}
