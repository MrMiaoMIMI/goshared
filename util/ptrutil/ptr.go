package ptrutil

// Of returns a pointer to the given value.
func Of[T any](v T) *T {
	return &v
}

// Value returns the value pointed to by p, or the zero value of T if p is nil.
func Value[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// ValueOr returns the value pointed to by p, or fallback if p is nil.
func ValueOr[T any](p *T, fallback T) T {
	if p == nil {
		return fallback
	}
	return *p
}

// IsNil returns true if the pointer is nil.
func IsNil[T any](p *T) bool {
	return p == nil
}

// Equal returns true if both pointers are nil, or both point to equal values.
func Equal[T comparable](a, b *T) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
