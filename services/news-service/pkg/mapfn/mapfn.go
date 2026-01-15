package mapfn

// ConvertSlice converts a slice of type T to a slice of type R using the provided function
func ConvertSlice[T any, R any](input []T, fn func(T) R) []R {
	result := make([]R, len(input))
	for i, v := range input {
		result[i] = fn(v)
	}
	return result
}

// FilterSlice filters a slice based on the provided predicate function
func FilterSlice[T any](input []T, predicate func(T) bool) []T {
	result := make([]T, 0)
	for _, v := range input {
		if predicate(v) {
			result = append(result, v)
		}
	}
	return result
}

// MapSlice applies function to each element and returns new slice (alias for ConvertSlice)
func MapSlice[T any, R any](input []T, fn func(T) R) []R {
	return ConvertSlice(input, fn)
}
