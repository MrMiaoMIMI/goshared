// Package env provides typed environment variable helpers.
package env

import (
	"os"
	"strings"
	"time"

	"github.com/MrMiaoMIMI/goshared/util/convutil"
)

// Value returns the raw value of key, or an empty string when key is not set.
func Value(key string) string {
	return os.Getenv(key)
}

// Get reads key and converts it into T. It returns fallback when key is unset
// or conversion fails.
func Get[T convutil.Scalar](key string, fallback T) T {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	return convutil.ToOr[T](v, fallback)
}

// Must returns key's raw value, panicking when key is unset or empty.
func Must(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("env: required environment variable " + key + " is not set")
	}
	return v
}

// Duration reads key as a time.Duration. It expects Go duration syntax such as
// "500ms", "5s", or "2h".
func Duration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

// Slice reads key, splits it by separator, trims whitespace, and removes empty
// values. It returns fallback when key is unset or empty.
func Slice(key, separator string, fallback []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	parts := strings.Split(v, separator)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// IsSet returns true when key is present, even if its value is empty.
func IsSet(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

// IsProd returns true if GO_ENV, APP_ENV, or ENV is "production" or "prod".
func IsProd() bool {
	return matchMode("production", "prod")
}

// IsDev returns true if GO_ENV, APP_ENV, or ENV is "development" or "dev".
func IsDev() bool {
	return matchMode("development", "dev")
}

func matchMode(values ...string) bool {
	for _, key := range []string{"GO_ENV", "APP_ENV", "ENV"} {
		current := strings.ToLower(os.Getenv(key))
		for _, value := range values {
			if current == value {
				return true
			}
		}
	}
	return false
}
