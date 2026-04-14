// Package env provides utility functions for reading environment variables with type safety.
package env

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Get returns the value of the environment variable or empty string.
func Get(key string) string {
	return os.Getenv(key)
}

// GetOrDefault returns the value of the environment variable or the default value.
func GetOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// MustGet returns the value of the environment variable or panics if not set.
func MustGet(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("env: required environment variable " + key + " is not set")
	}
	return v
}

// GetInt returns the environment variable as an int, or defaultVal on error.
func GetInt(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

// GetInt64 returns the environment variable as an int64, or defaultVal on error.
func GetInt64(key string, defaultVal int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return defaultVal
	}
	return n
}

// GetFloat64 returns the environment variable as a float64, or defaultVal on error.
func GetFloat64(key string, defaultVal float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return defaultVal
	}
	return f
}

// GetBool returns the environment variable as a bool, or defaultVal on error.
// Truthy: "true", "1", "yes", "on" (case-insensitive).
// Falsy: "false", "0", "no", "off", "" (case-insensitive).
func GetBool(key string, defaultVal bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	switch strings.ToLower(v) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultVal
	}
}

// GetDuration returns the environment variable as a time.Duration, or defaultVal on error.
// The value should be in Go duration format (e.g., "5s", "1m30s", "2h").
func GetDuration(key string, defaultVal time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return defaultVal
	}
	return d
}

// GetSlice returns the environment variable split by separator.
// Returns defaultVal if the variable is not set.
func GetSlice(key, separator string, defaultVal []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	parts := strings.Split(v, separator)
	var result []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// IsSet returns true if the environment variable is set (even if empty).
func IsSet(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}

// IsProd returns true if the environment looks like production.
// Checks GO_ENV, APP_ENV, ENV for "production" or "prod".
func IsProd() bool {
	for _, key := range []string{"GO_ENV", "APP_ENV", "ENV"} {
		v := strings.ToLower(os.Getenv(key))
		if v == "production" || v == "prod" {
			return true
		}
	}
	return false
}

// IsDev returns true if the environment looks like development.
func IsDev() bool {
	for _, key := range []string{"GO_ENV", "APP_ENV", "ENV"} {
		v := strings.ToLower(os.Getenv(key))
		if v == "development" || v == "dev" {
			return true
		}
	}
	return false
}
