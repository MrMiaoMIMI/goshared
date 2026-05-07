package cachehelper

import (
	"fmt"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/MrMiaoMIMI/goshared/cache/internal/cachesp"
)

// NewCache creates a Cache from the user-facing cache configuration.
func NewCache(cfg cachespi.CacheConfig) (cachespi.Cache, error) {
	switch cfg.Backend {
	case cachespi.BackendInMemory:
		return NewInMemCache(cfg.InMemory)
	case cachespi.BackendRedis:
		return NewRedisCache(cfg.Redis)
	case "":
		return nil, fmt.Errorf("cache: backend is required")
	default:
		return nil, fmt.Errorf("cache: unsupported backend %q", cfg.Backend)
	}
}

// NewRedisCache creates a Cache backed by Redis from cfg.
//
// Defaults when cfg leaves a field as its zero value:
//   - address: localhost:6379
//   - password: empty
//   - database: 0
//   - pool size: 10
//   - minimum idle connections: 2
//   - dial timeout: 5 seconds
//   - read timeout: 3 seconds
//   - write timeout: 3 seconds
//   - default TTL: 5 minutes
//   - codec: cachespi.JSONCodec
//
// NewRedisCache creates and owns a Redis client; cache.Close(ctx) closes that
// client.
//
// Example:
//
//	cache, err := cachehelper.NewRedisCache(cachespi.RedisConfig{
//	    Addr:       "localhost:6379",
//	    Password:   "secret",
//	    DB:         0,
//	    DefaultTTL: 10 * time.Minute,
//	    Codec:      cachespi.CodecJSON,
//	})
func NewRedisCache(cfg cachespi.RedisConfig) (cachespi.Cache, error) {
	codec, err := codecFromName(cfg.Codec, nil)
	if err != nil {
		return nil, err
	}
	return cachesp.NewRedisCache(cachesp.RedisConfig{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		DefaultTTL:   cfg.DefaultTTL,
		Codec:        codec,
	})
}
