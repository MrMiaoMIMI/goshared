package random

import (
	"math/rand"
	"time"
)

var (
	randInstance = rand.New(rand.NewSource(time.Now().UnixNano()))
)

func GetRand() *rand.Rand {
	return randInstance
}

// PickOne picks one random element from items.
// Panics if items is empty.
func PickOne[T any](items []T) T {
	return items[GetRand().Intn(len(items))]
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
	n := min + GetRand().Intn(max-min+1)
	shuffled := make([]T, len(items))
	copy(shuffled, items)
	GetRand().Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return shuffled[:n]
}
