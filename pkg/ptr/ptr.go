package ptr

func Assign[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}
