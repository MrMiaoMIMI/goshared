package cachehelper

import (
	"fmt"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/MrMiaoMIMI/goshared/cache/internal/cachesp"
)

// NewInMemCache creates a Cache backed by Ristretto in local memory from cfg.
//
// Defaults when cfg leaves a field as its zero value:
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
//	cache, err := cachehelper.NewInMemCache(cachespi.InMemoryConfig{
//	    DefaultTTL:  10 * time.Minute,
//	    MaxCost:     1 << 28, // 256 MB
//	    NumCounters: 1e6,
//	})
func NewInMemCache(cfg cachespi.InMemoryConfig) (cachespi.Cache, error) {
	codec, err := codecFromName(cfg.Codec, nil)
	if err != nil {
		return nil, err
	}
	return cachesp.NewRistrettoCache(cachesp.RistrettoConfig{
		NumCounters: cfg.NumCounters,
		MaxCost:     cfg.MaxCost,
		BufferItems: cfg.BufferItems,
		DefaultTTL:  cfg.DefaultTTL,
		Codec:       codec,
	})
}

func codecFromName(name cachespi.CodecName, defaultCodec cachespi.Codec) (cachespi.Codec, error) {
	switch name {
	case cachespi.CodecDefault:
		return defaultCodec, nil
	case cachespi.CodecJSON:
		return cachespi.JSONCodec{}, nil
	default:
		return nil, fmt.Errorf("cache: unsupported codec %q", name)
	}
}
