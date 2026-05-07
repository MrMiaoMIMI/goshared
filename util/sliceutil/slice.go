// Package sliceutil provides generic helpers for common slice transformations.
package sliceutil

import (
	"cmp"
	"slices"
)

// Map transforms each element.
func Map[T any, U any](in []T, fn func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}

// FlatMap maps each element to a slice and flattens the results.
func FlatMap[T any, U any](in []T, fn func(T) []U) []U {
	out := make([]U, 0)
	for _, v := range in {
		out = append(out, fn(v)...)
	}
	return out
}

// Filter returns elements that satisfy fn.
func Filter[T any](in []T, fn func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		if fn(v) {
			out = append(out, v)
		}
	}
	return out
}

// Partition splits in into matching and non-matching elements.
func Partition[T any](in []T, fn func(T) bool) (matched []T, unmatched []T) {
	for _, v := range in {
		if fn(v) {
			matched = append(matched, v)
			continue
		}
		unmatched = append(unmatched, v)
	}
	return matched, unmatched
}

// Reduce folds in into one value.
func Reduce[T any, U any](in []T, initial U, fn func(U, T) U) U {
	acc := initial
	for _, v := range in {
		acc = fn(acc, v)
	}
	return acc
}

// Any reports whether any element satisfies fn.
func Any[T any](in []T, fn func(T) bool) bool {
	for _, v := range in {
		if fn(v) {
			return true
		}
	}
	return false
}

// All reports whether all elements satisfy fn.
func All[T any](in []T, fn func(T) bool) bool {
	for _, v := range in {
		if !fn(v) {
			return false
		}
	}
	return true
}

// None reports whether no elements satisfy fn.
func None[T any](in []T, fn func(T) bool) bool {
	return !Any(in, fn)
}

// Count returns the number of elements satisfying fn.
func Count[T any](in []T, fn func(T) bool) int {
	count := 0
	for _, v := range in {
		if fn(v) {
			count++
		}
	}
	return count
}

// Find returns the first element satisfying fn.
func Find[T any](in []T, fn func(T) bool) (T, bool) {
	for _, v := range in {
		if fn(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// First returns the first element and whether it exists.
func First[T any](in []T) (T, bool) {
	if len(in) == 0 {
		var zero T
		return zero, false
	}
	return in[0], true
}

// FirstOr returns the first element, or fallback when in is empty.
func FirstOr[T any](in []T, fallback T) T {
	if v, ok := First(in); ok {
		return v
	}
	return fallback
}

// Last returns the last element and whether it exists.
func Last[T any](in []T) (T, bool) {
	if len(in) == 0 {
		var zero T
		return zero, false
	}
	return in[len(in)-1], true
}

// LastOr returns the last element, or fallback when in is empty.
func LastOr[T any](in []T, fallback T) T {
	if v, ok := Last(in); ok {
		return v
	}
	return fallback
}

// Unique returns a new slice with duplicate values removed, preserving order.
func Unique[T comparable](in []T) []T {
	seen := make(map[T]struct{}, len(in))
	out := make([]T, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// UniqueBy returns a new slice with duplicate keys removed, preserving order.
func UniqueBy[T any, K comparable](in []T, keyFn func(T) K) []T {
	seen := make(map[K]struct{}, len(in))
	out := make([]T, 0, len(in))
	for _, v := range in {
		key := keyFn(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	return out
}

// GroupBy groups elements by key.
func GroupBy[T any, K comparable](in []T, keyFn func(T) K) map[K][]T {
	out := make(map[K][]T)
	for _, v := range in {
		key := keyFn(v)
		out[key] = append(out[key], v)
	}
	return out
}

// ToMap converts a slice to a map using key and value functions.
func ToMap[T any, K comparable, V any](in []T, keyFn func(T) K, valueFn func(T) V) map[K]V {
	out := make(map[K]V, len(in))
	for _, v := range in {
		out[keyFn(v)] = valueFn(v)
	}
	return out
}

// Flatten merges nested slices into one slice.
func Flatten[T any](in [][]T) []T {
	total := 0
	for _, part := range in {
		total += len(part)
	}
	out := make([]T, 0, total)
	for _, part := range in {
		out = append(out, part...)
	}
	return out
}

// Chunk splits in into chunks of size. It returns nil when size <= 0.
func Chunk[T any](in []T, size int) [][]T {
	if size <= 0 {
		return nil
	}
	out := make([][]T, 0, (len(in)+size-1)/size)
	for i := 0; i < len(in); i += size {
		end := min(i+size, len(in))
		out = append(out, in[i:end])
	}
	return out
}

// Reverse returns a reversed copy.
func Reverse[T any](in []T) []T {
	out := slices.Clone(in)
	slices.Reverse(out)
	return out
}

// Sort returns a sorted copy.
func Sort[T cmp.Ordered](in []T) []T {
	out := slices.Clone(in)
	slices.Sort(out)
	return out
}

// SortBy returns a sorted copy using cmpFn.
func SortBy[T any](in []T, cmpFn func(a, b T) int) []T {
	out := slices.Clone(in)
	slices.SortFunc(out, cmpFn)
	return out
}

// SortStableBy returns a stably sorted copy using cmpFn.
func SortStableBy[T any](in []T, cmpFn func(a, b T) int) []T {
	out := slices.Clone(in)
	slices.SortStableFunc(out, cmpFn)
	return out
}

// Difference returns values from a that are not present in b.
func Difference[T comparable](a, b []T) []T {
	excluded := make(map[T]struct{}, len(b))
	for _, v := range b {
		excluded[v] = struct{}{}
	}
	out := make([]T, 0, len(a))
	for _, v := range a {
		if _, ok := excluded[v]; !ok {
			out = append(out, v)
		}
	}
	return out
}

// Intersect returns unique values that appear in both slices.
func Intersect[T comparable](a, b []T) []T {
	inB := make(map[T]struct{}, len(b))
	for _, v := range b {
		inB[v] = struct{}{}
	}
	out := make([]T, 0)
	seen := make(map[T]struct{})
	for _, v := range a {
		if _, ok := inB[v]; !ok {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// Union returns unique values from both slices, preserving first appearance.
func Union[T comparable](a, b []T) []T {
	out := make([]T, 0, len(a)+len(b))
	seen := make(map[T]struct{}, len(a)+len(b))
	for _, slice := range [][]T{a, b} {
		for _, v := range slice {
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}
