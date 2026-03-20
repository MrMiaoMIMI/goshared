package cachehelper

import (
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/MrMiaoMIMI/goshared/cache/internal/cachesp"
)

// CacheOption is a type alias so callers don't need to import the internal package.
type CacheOption = cachesp.CacheOption

// NewInMemCache creates a new Cache instance backed by ristretto.
func NewInMemCache(opts ...CacheOption) cachespi.Cache {
	return cachesp.NewRistrettoCache(opts...)
}

func WithDefaultTTL(d time.Duration) CacheOption {
	return cachesp.WithDefaultTTL(d)
}
