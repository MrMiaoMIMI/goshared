package cachehelper

import (
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/MrMiaoMIMI/goshared/cache/internal/cachesp"
)

// NewRedisCache creates a Cache backed by Redis.
//
// Defaults when no option is provided:
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
//	cache, err := cachehelper.NewRedisCache(
//	    cachehelper.WithRedisAddr("localhost:6379"),
//	    cachehelper.WithRedisPassword("secret"),
//	    cachehelper.WithRedisDB(0),
//	    cachehelper.WithRedisDefaultTTL(10 * time.Minute),
//	    cachehelper.WithRedisCodec(cachespi.JSONCodec{}),
//	)
func NewRedisCache(opts ...RedisOption) (cachespi.Cache, error) {
	internalOpts := &redisOptions{
		internal: make([]cachesp.RedisOption, 0, len(opts)),
	}
	for _, opt := range opts {
		if opt != nil {
			opt.applyRedisOption(internalOpts)
		}
	}
	return cachesp.NewRedisCache(internalOpts.internal...)
}

// WithRedisAddr sets the Redis server address in host:port form.
//
// If unset, the default address is localhost:6379.
func WithRedisAddr(addr string) RedisOption {
	return redisCacheOption(cachesp.WithRedisAddr(addr))
}

// WithRedisPassword sets the Redis password.
//
// If unset, no password is used.
func WithRedisPassword(password string) RedisOption {
	return redisCacheOption(cachesp.WithRedisPassword(password))
}

// WithRedisDB selects the Redis database index.
//
// If unset, database 0 is used.
func WithRedisDB(db int) RedisOption {
	return redisCacheOption(cachesp.WithRedisDB(db))
}

// WithRedisPoolSize sets the maximum number of connections in the pool.
//
// If unset, the default pool size is 10.
func WithRedisPoolSize(size int) RedisOption {
	return redisCacheOption(cachesp.WithRedisPoolSize(size))
}

// WithRedisMinIdleConns sets the minimum number of idle connections.
//
// If unset, the default is 2 idle connections.
func WithRedisMinIdleConns(n int) RedisOption {
	return redisCacheOption(cachesp.WithRedisMinIdleConns(n))
}

// WithRedisDialTimeout sets the timeout for establishing new connections.
//
// If unset, the default is 5 seconds.
func WithRedisDialTimeout(d time.Duration) RedisOption {
	return redisCacheOption(cachesp.WithRedisDialTimeout(d))
}

// WithRedisReadTimeout sets the timeout for socket reads.
//
// If unset, the default is 3 seconds.
func WithRedisReadTimeout(d time.Duration) RedisOption {
	return redisCacheOption(cachesp.WithRedisReadTimeout(d))
}

// WithRedisWriteTimeout sets the timeout for socket writes.
//
// If unset, the default is 3 seconds.
func WithRedisWriteTimeout(d time.Duration) RedisOption {
	return redisCacheOption(cachesp.WithRedisWriteTimeout(d))
}

// WithRedisDefaultTTL sets the default TTL used when a cache operation receives
// cachespi.DefaultExpiration.
//
// If unset, the default TTL is 5 minutes. Use cachespi.NoExpiration to make
// DefaultExpiration mean no expiration for this cache instance. Other negative
// values cause NewRedisCache to return cachespi.ErrInvalidExpiration.
func WithRedisDefaultTTL(d time.Duration) RedisOption {
	return redisCacheOption(cachesp.WithRedisDefaultTTL(d))
}

// WithRedisCodec sets the codec used to serialize Redis cache values.
//
// If unset, cachespi.JSONCodec is used. Redis stores bytes across a
// network/process boundary, so reference semantics are not supported.
func WithRedisCodec(codec cachespi.Codec) RedisOption {
	return redisCacheOption(cachesp.WithRedisCodec(codec))
}

func redisCacheOption(opt cachesp.RedisOption) RedisOption {
	return redisOptionFunc(func(opts *redisOptions) {
		opts.internal = append(opts.internal, opt)
	})
}

// RedisOption configures a Redis cache.
type RedisOption interface {
	applyRedisOption(*redisOptions)
}

type redisOptionFunc func(*redisOptions)

func (f redisOptionFunc) applyRedisOption(opts *redisOptions) {
	f(opts)
}

type redisOptions struct {
	internal []cachesp.RedisOption
}
