package cachehelper

import (
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/MrMiaoMIMI/goshared/cache/internal/cachesp"
)

// CacheOption is a type alias so callers don't need to import the internal package.
type CacheOption = cachesp.CacheOption

// NewInMemCache creates a new Cache instance backed by ristretto.
//
// Example:
//
//	cache := cachehelper.NewInMemCache(
//	    cachehelper.WithDefaultTTL(10 * time.Minute),
//	    cachehelper.WithMaxCost(1 << 28), // 256 MB
//	    cachehelper.WithNumCounters(1e6),
//	)
func NewInMemCache(opts ...CacheOption) cachespi.Cache {
	return cachesp.NewRistrettoCache(opts...)
}

// WithDefaultTTL sets the default TTL for cache entries.
func WithDefaultTTL(d time.Duration) CacheOption {
	return cachesp.WithDefaultTTL(d)
}

// WithNumCounters sets the number of counters (keys) used by the admission policy.
// Recommended: 10x the expected number of unique keys.
func WithNumCounters(n int64) CacheOption {
	return cachesp.WithNumCounters(n)
}

// WithMaxCost sets the maximum cost (memory in bytes) of the cache.
func WithMaxCost(n int64) CacheOption {
	return cachesp.WithMaxCost(n)
}

// WithBufferItems sets the number of buffer items for the ristretto ring buffer.
func WithBufferItems(n int64) CacheOption {
	return cachesp.WithBufferItems(n)
}
