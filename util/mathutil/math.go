// Package mathutil provides generic numeric utility functions for comparable and ordered types.
package mathutil

import (
	"cmp"
	"math"
)

// Number is a constraint for all numeric types.
type Number interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Min returns the smaller of two ordered values.
func Min[T cmp.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of two ordered values.
func Max[T cmp.Ordered](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// Clamp restricts val to the range [lo, hi].
func Clamp[T cmp.Ordered](val, lo, hi T) T {
	if val < lo {
		return lo
	}
	if val > hi {
		return hi
	}
	return val
}

// Abs returns the absolute value of a signed number.
func Abs[T ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64](v T) T {
	if v < 0 {
		return -v
	}
	return v
}

// Sum returns the sum of all elements in the slice.
func Sum[T Number](values []T) T {
	var total T
	for _, v := range values {
		total += v
	}
	return total
}

// Average returns the arithmetic mean of the values.
// Returns 0 for an empty slice.
func Average[T Number](values []T) float64 {
	if len(values) == 0 {
		return 0
	}
	var total float64
	for _, v := range values {
		total += float64(v)
	}
	return total / float64(len(values))
}

// MinSlice returns the minimum value in the slice.
// Panics on empty slice.
func MinSlice[T cmp.Ordered](values []T) T {
	if len(values) == 0 {
		panic("mathutil: MinSlice called on empty slice")
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

// MaxSlice returns the maximum value in the slice.
// Panics on empty slice.
func MaxSlice[T cmp.Ordered](values []T) T {
	if len(values) == 0 {
		panic("mathutil: MaxSlice called on empty slice")
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// InRange checks whether val is within [lo, hi] (inclusive).
func InRange[T cmp.Ordered](val, lo, hi T) bool {
	return val >= lo && val <= hi
}

// Percentage calculates (part / total * 100) safely, returning 0 if total is 0.
func Percentage(part, total float64) float64 {
	if total == 0 {
		return 0
	}
	return (part / total) * 100
}

// RoundTo rounds a float64 to the given number of decimal places.
func RoundTo(val float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(val*pow) / pow
}

// CeilDiv performs integer division rounding up (ceiling division).
// Both a and b must be non-negative (a >= 0, b > 0).
// For negative values, use standard integer division instead.
func CeilDiv[T ~int | ~int32 | ~int64](a, b T) T {
	if b == 0 {
		panic("mathutil: division by zero")
	}
	if a < 0 || b < 0 {
		panic("mathutil: CeilDiv requires non-negative arguments")
	}
	return (a + b - 1) / b
}

// DivMod returns quotient and remainder.
func DivMod[T ~int | ~int32 | ~int64](a, b T) (T, T) {
	if b == 0 {
		panic("mathutil: division by zero")
	}
	return a / b, a % b
}

// SafeDiv divides a by b, returning defaultVal if b is zero.
func SafeDiv[T Number](a, b, defaultVal T) T {
	if b == 0 {
		return defaultVal
	}
	return a / b
}
