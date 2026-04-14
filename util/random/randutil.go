package random

import (
	"math/rand/v2"
)

// PickOne picks one random element from items.
// Panics if items is empty.
func PickOne[T any](items []T) T {
	if len(items) == 0 {
		panic("random: PickOne called on empty slice")
	}
	return items[rand.IntN(len(items))]
}

// PickOneOr picks a random element from items, returning fallback if items is empty.
func PickOneOr[T any](items []T, fallback T) T {
	if len(items) == 0 {
		return fallback
	}
	return items[rand.IntN(len(items))]
}

// Float64 returns a random float64 in [0.0, 1.0).
func Float64() float64 {
	return rand.Float64()
}

// Float64Range returns a random float64 in [min, max).
func Float64Range(min, max float64) float64 {
	if min > max {
		min, max = max, min
	}
	return min + rand.Float64()*(max-min)
}

// Bool returns a random boolean.
func Bool() bool {
	return rand.IntN(2) == 1
}

// String generates a random string of the given length using the specified charset.
// If charset is empty, uses alphanumeric characters.
// Returns empty string if length <= 0.
func String(length int, charset string) string {
	if length <= 0 {
		return ""
	}
	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.IntN(len(charset))]
	}
	return string(b)
}

// PickN picks between min and max (inclusive) unique elements from items.
// If max > len(items), it is clamped to len(items).
func PickN[T any](items []T, min, max int) []T {
	if max > len(items) {
		max = len(items)
	}
	if min > max {
		min = max
	}
	n := min + rand.IntN(max-min+1)
	shuffled := make([]T, len(items))
	copy(shuffled, items)
	rand.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return shuffled[:n]
}

// Shuffle returns a new slice with elements in random order.
// The original slice is not modified.
func Shuffle[T any](items []T) []T {
	result := make([]T, len(items))
	copy(result, items)
	rand.Shuffle(len(result), func(i, j int) { result[i], result[j] = result[j], result[i] })
	return result
}

// IntN returns a random int in [0, n).
func IntN(n int) int {
	return rand.IntN(n)
}

// IntRange returns a random int in [min, max].
func IntRange(min, max int) int {
	if min > max {
		min, max = max, min
	}
	return min + rand.IntN(max-min+1)
}

// WeightedPick picks one element from items based on weights.
// Panics if items and weights have different lengths or if all weights are zero.
func WeightedPick[T any](items []T, weights []float64) T {
	if len(items) != len(weights) {
		panic("random: items and weights must have the same length")
	}
	var total float64
	for _, w := range weights {
		total += w
	}
	if total == 0 {
		panic("random: total weight must be positive")
	}
	r := rand.Float64() * total
	for i, w := range weights {
		r -= w
		if r <= 0 {
			return items[i]
		}
	}
	return items[len(items)-1]
}

// Sample picks n unique elements from items using reservoir sampling.
// If n >= len(items), returns a shuffled copy of items.
func Sample[T any](items []T, n int) []T {
	if n >= len(items) {
		return Shuffle(items)
	}
	result := make([]T, n)
	copy(result, items[:n])
	for i := n; i < len(items); i++ {
		j := rand.IntN(i + 1)
		if j < n {
			result[j] = items[i]
		}
	}
	return result
}
