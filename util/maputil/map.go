// Package maputil provides generic helpers for map transformations.
package maputil

// Keys returns all keys from m.
func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Values returns all values from m.
func Values[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

// GetOr returns m[key], or fallback when key does not exist.
func GetOr[K comparable, V any](m map[K]V, key K, fallback V) V {
	if v, ok := m[key]; ok {
		return v
	}
	return fallback
}

// Merge merges maps into a new map. Later maps override earlier maps.
func Merge[K comparable, V any](maps ...map[K]V) map[K]V {
	size := 0
	for _, m := range maps {
		size += len(m)
	}
	out := make(map[K]V, size)
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// Filter returns a new map containing entries that satisfy fn.
func Filter[K comparable, V any](m map[K]V, fn func(K, V) bool) map[K]V {
	out := make(map[K]V)
	for k, v := range m {
		if fn(k, v) {
			out[k] = v
		}
	}
	return out
}

// MapValues transforms map values while preserving keys.
func MapValues[K comparable, V any, U any](m map[K]V, fn func(V) U) map[K]U {
	out := make(map[K]U, len(m))
	for k, v := range m {
		out[k] = fn(v)
	}
	return out
}

// Pick returns a new map containing only selected keys.
func Pick[K comparable, V any](m map[K]V, keys ...K) map[K]V {
	out := make(map[K]V, len(keys))
	for _, k := range keys {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	return out
}

// Omit returns a new map without selected keys.
func Omit[K comparable, V any](m map[K]V, keys ...K) map[K]V {
	excluded := make(map[K]struct{}, len(keys))
	for _, k := range keys {
		excluded[k] = struct{}{}
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		if _, ok := excluded[k]; !ok {
			out[k] = v
		}
	}
	return out
}
