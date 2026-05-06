package cachehelper

import (
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/MrMiaoMIMI/goshared/cache/internal/cachesp"
)

// NewInMemCache creates a Cache backed by Ristretto in local memory.
//
// Defaults when no option is provided:
//   - default TTL: 5 minutes
//   - num counters: 1e7
//   - max cost: 1 GiB
//   - buffer items: 64
//   - value semantics: reference semantics, no codec
//
// The returned cache should be closed when it is no longer used.
//
// Example:
//
//	cache, err := cachehelper.NewInMemCache(
//	    cachehelper.WithInMemDefaultTTL(10 * time.Minute),
//	    cachehelper.WithInMemMaxCost(1 << 28), // 256 MB
//	    cachehelper.WithInMemNumCounters(1e6),
//	)
func NewInMemCache(opts ...InMemCacheOption) (cachespi.Cache, error) {
	internalOpts := &inMemOptions{
		internal: make([]cachesp.RistrettoOption, 0, len(opts)),
	}
	for _, opt := range opts {
		if opt != nil {
			opt.applyInMemOption(internalOpts)
		}
	}
	return cachesp.NewRistrettoCache(internalOpts.internal...)
}

// WithInMemDefaultTTL sets the default TTL used when a cache operation receives
// cachespi.DefaultExpiration.
//
// If unset, the default TTL is 5 minutes. Use cachespi.NoExpiration to make
// DefaultExpiration mean no expiration for this cache instance. Other negative
// values cause NewInMemCache to return cachespi.ErrInvalidExpiration.
func WithInMemDefaultTTL(d time.Duration) InMemCacheOption {
	return inMemOption(cachesp.WithRistrettoDefaultTTL(d))
}

// WithInMemNumCounters sets the number of counters (keys) used by the admission policy.
//
// If unset, the default is 1e7. Ristretto recommends roughly 10x the expected
// number of unique keys.
func WithInMemNumCounters(n int64) InMemCacheOption {
	return inMemOption(cachesp.WithRistrettoNumCounters(n))
}

// WithInMemMaxCost sets the maximum total cost of the cache.
//
// If unset, the default is 1 GiB. Current cache writes use cost 1 per key, so
// this value acts as the maximum number of admitted items unless the internal
// implementation starts using custom costs.
func WithInMemMaxCost(n int64) InMemCacheOption {
	return inMemOption(cachesp.WithRistrettoMaxCost(n))
}

// WithInMemBufferItems sets the number of buffer items for the ristretto ring buffer.
//
// If unset, the default is 64.
func WithInMemBufferItems(n int64) InMemCacheOption {
	return inMemOption(cachesp.WithRistrettoBufferItems(n))
}

// WithInMemCodec sets the codec used for in-memory cache values.
//
// If unset, in-memory cache stores Go values directly by reference. That means
// mutating cached pointer, map, or slice values can affect later reads. Use a
// codec such as cachespi.JSONCodec to store serialized bytes and get Redis-like
// copy semantics.
func WithInMemCodec(codec cachespi.Codec) InMemCacheOption {
	return inMemOption(cachesp.WithRistrettoCodec(codec))
}

func inMemOption(opt cachesp.RistrettoOption) InMemCacheOption {
	return inMemOptionFunc(func(opts *inMemOptions) {
		opts.internal = append(opts.internal, opt)
	})
}

// InMemCacheOption configures an in-memory cache.
type InMemCacheOption interface {
	applyInMemOption(*inMemOptions)
}

type inMemOptionFunc func(*inMemOptions)

func (f inMemOptionFunc) applyInMemOption(opts *inMemOptions) {
	f(opts)
}

type inMemOptions struct {
	internal []cachesp.RistrettoOption
}
