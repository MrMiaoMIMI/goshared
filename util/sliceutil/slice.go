// Package sliceutil provides generic utility functions for working with slices.
package sliceutil

import (
	"cmp"
	"slices"
)

// Contains returns true if the slice contains the target value.
func Contains[T comparable](slice []T, target T) bool {
	for _, v := range slice {
		if v == target {
			return true
		}
	}
	return false
}

// ContainsFunc returns true if any element satisfies the predicate.
func ContainsFunc[T any](slice []T, fn func(T) bool) bool {
	for _, v := range slice {
		if fn(v) {
			return true
		}
	}
	return false
}

// Map transforms a slice using the given function.
func Map[T any, U any](slice []T, fn func(T) U) []U {
	result := make([]U, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

// Filter returns elements that satisfy the predicate.
func Filter[T any](slice []T, fn func(T) bool) []T {
	var result []T
	for _, v := range slice {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

// Reduce reduces a slice to a single value using the accumulator function.
func Reduce[T any, U any](slice []T, initial U, fn func(U, T) U) U {
	acc := initial
	for _, v := range slice {
		acc = fn(acc, v)
	}
	return acc
}

// Unique returns a new slice with duplicate elements removed.
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{}, len(slice))
	var result []T
	for _, v := range slice {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// UniqueBy returns a new slice with duplicates removed based on a key function.
func UniqueBy[T any, K comparable](slice []T, keyFn func(T) K) []T {
	seen := make(map[K]struct{}, len(slice))
	var result []T
	for _, v := range slice {
		key := keyFn(v)
		if _, ok := seen[key]; !ok {
			seen[key] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

// GroupBy groups slice elements by a key function.
func GroupBy[T any, K comparable](slice []T, keyFn func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, v := range slice {
		key := keyFn(v)
		result[key] = append(result[key], v)
	}
	return result
}

// Flatten merges nested slices into a single slice.
func Flatten[T any](slices [][]T) []T {
	var result []T
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// Chunk splits a slice into chunks of the given size.
func Chunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		return nil
	}
	var chunks [][]T
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// Reverse returns a new slice with elements in reverse order.
func Reverse[T any](slice []T) []T {
	result := make([]T, len(slice))
	for i, v := range slice {
		result[len(slice)-1-i] = v
	}
	return result
}

// First returns the first element, or the zero value if empty.
func First[T any](slice []T) T {
	if len(slice) == 0 {
		var zero T
		return zero
	}
	return slice[0]
}

// Last returns the last element, or the zero value if empty.
func Last[T any](slice []T) T {
	if len(slice) == 0 {
		var zero T
		return zero
	}
	return slice[len(slice)-1]
}

// FindFunc returns the first element satisfying the predicate and true,
// or the zero value and false if none found.
func FindFunc[T any](slice []T, fn func(T) bool) (T, bool) {
	for _, v := range slice {
		if fn(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// IndexOf returns the index of the first occurrence of target, or -1.
func IndexOf[T comparable](slice []T, target T) int {
	for i, v := range slice {
		if v == target {
			return i
		}
	}
	return -1
}

// ToMap converts a slice to a map using key and value functions.
func ToMap[T any, K comparable, V any](slice []T, keyFn func(T) K, valFn func(T) V) map[K]V {
	result := make(map[K]V, len(slice))
	for _, v := range slice {
		result[keyFn(v)] = valFn(v)
	}
	return result
}

// Difference returns elements in a that are not in b.
func Difference[T comparable](a, b []T) []T {
	set := make(map[T]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}
	var result []T
	for _, v := range a {
		if _, ok := set[v]; !ok {
			result = append(result, v)
		}
	}
	return result
}

// Intersect returns elements common to both a and b.
func Intersect[T comparable](a, b []T) []T {
	set := make(map[T]struct{}, len(b))
	for _, v := range b {
		set[v] = struct{}{}
	}
	var result []T
	for _, v := range a {
		if _, ok := set[v]; ok {
			result = append(result, v)
		}
	}
	return Unique(result)
}

// Sort sorts a slice of ordered values in ascending order. Returns a new slice.
func Sort[T cmp.Ordered](s []T) []T {
	result := make([]T, len(s))
	copy(result, s)
	slices.Sort(result)
	return result
}

// SortBy sorts a slice using a custom comparison function. Returns a new slice.
func SortBy[T any](s []T, less func(a, b T) int) []T {
	result := make([]T, len(s))
	copy(result, s)
	slices.SortFunc(result, less)
	return result
}

// SortStableBy sorts a slice using a custom comparison function with stable ordering.
func SortStableBy[T any](s []T, less func(a, b T) int) []T {
	result := make([]T, len(s))
	copy(result, s)
	slices.SortStableFunc(result, less)
	return result
}

// FlatMap maps each element to a slice and flattens the results.
func FlatMap[T any, U any](slice []T, fn func(T) []U) []U {
	var result []U
	for _, v := range slice {
		result = append(result, fn(v)...)
	}
	return result
}

// Compact removes consecutive duplicate elements from a sorted slice.
func Compact[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return nil
	}
	result := []T{slice[0]}
	for i := 1; i < len(slice); i++ {
		if slice[i] != slice[i-1] {
			result = append(result, slice[i])
		}
	}
	return result
}

// ForEach calls fn for each element. Unlike Map, it doesn't return a result.
func ForEach[T any](slice []T, fn func(int, T)) {
	for i, v := range slice {
		fn(i, v)
	}
}

// Count returns the number of elements satisfying the predicate.
func Count[T any](slice []T, fn func(T) bool) int {
	n := 0
	for _, v := range slice {
		if fn(v) {
			n++
		}
	}
	return n
}

// Zip combines two slices into a slice of pairs.
// The result length equals the shorter of the two inputs.
func Zip[T any, U any](a []T, b []U) []Pair[T, U] {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	result := make([]Pair[T, U], n)
	for i := 0; i < n; i++ {
		result[i] = Pair[T, U]{First: a[i], Second: b[i]}
	}
	return result
}

// Pair holds two values of potentially different types.
type Pair[T any, U any] struct {
	First  T
	Second U
}

// Partition splits a slice into two: elements satisfying the predicate and elements that don't.
func Partition[T any](slice []T, fn func(T) bool) (pass []T, fail []T) {
	for _, v := range slice {
		if fn(v) {
			pass = append(pass, v)
		} else {
			fail = append(fail, v)
		}
	}
	return
}

// Union returns the set union of two slices (unique elements from both).
func Union[T comparable](a, b []T) []T {
	return Unique(append(append([]T{}, a...), b...))
}

// None returns true if no element satisfies the predicate.
func None[T any](slice []T, fn func(T) bool) bool {
	for _, v := range slice {
		if fn(v) {
			return false
		}
	}
	return true
}

// All returns true if all elements satisfy the predicate.
func All[T any](slice []T, fn func(T) bool) bool {
	for _, v := range slice {
		if !fn(v) {
			return false
		}
	}
	return true
}
