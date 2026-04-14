package cachehelper

import (
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/MrMiaoMIMI/goshared/cache/internal/cachesp"
	"github.com/redis/go-redis/v9"
)

// RedisOption configures the Redis cache provider.
type RedisOption = cachesp.RedisOption

// NewRedisCache creates a new Cache instance backed by Redis.
//
// Example:
//
//	cache := cachehelper.NewRedisCache(
//	    cachehelper.WithRedisAddr("localhost:6379"),
//	    cachehelper.WithRedisPassword("secret"),
//	    cachehelper.WithRedisDB(0),
//	    cachehelper.WithRedisDefaultTTL(10 * time.Minute),
//	)
func NewRedisCache(opts ...RedisOption) cachespi.Cache {
	return cachesp.NewRedisCache(opts...)
}

// WithRedisAddr sets the Redis server address (host:port).
func WithRedisAddr(addr string) RedisOption {
	return cachesp.WithRedisAddr(addr)
}

// WithRedisPassword sets the Redis password.
func WithRedisPassword(password string) RedisOption {
	return cachesp.WithRedisPassword(password)
}

// WithRedisDB selects the Redis database index.
func WithRedisDB(db int) RedisOption {
	return cachesp.WithRedisDB(db)
}

// WithRedisPoolSize sets the maximum number of connections in the pool.
func WithRedisPoolSize(size int) RedisOption {
	return cachesp.WithRedisPoolSize(size)
}

// WithRedisMinIdleConns sets the minimum number of idle connections.
func WithRedisMinIdleConns(n int) RedisOption {
	return cachesp.WithRedisMinIdleConns(n)
}

// WithRedisDialTimeout sets the timeout for establishing new connections.
func WithRedisDialTimeout(d time.Duration) RedisOption {
	return cachesp.WithRedisDialTimeout(d)
}

// WithRedisReadTimeout sets the timeout for socket reads.
func WithRedisReadTimeout(d time.Duration) RedisOption {
	return cachesp.WithRedisReadTimeout(d)
}

// WithRedisWriteTimeout sets the timeout for socket writes.
func WithRedisWriteTimeout(d time.Duration) RedisOption {
	return cachesp.WithRedisWriteTimeout(d)
}

// WithRedisDefaultTTL sets the default TTL for cache entries.
func WithRedisDefaultTTL(d time.Duration) RedisOption {
	return cachesp.WithRedisDefaultTTL(d)
}

// WithRedisClient allows injecting a pre-configured redis.UniversalClient.
// When set, all connection options (addr, password, etc.) are ignored.
// This is useful for Redis Cluster or Sentinel setups.
//
// Example (Cluster):
//
//	cluster := redis.NewClusterClient(&redis.ClusterOptions{
//	    Addrs: []string{"node1:6379", "node2:6379", "node3:6379"},
//	})
//	cache := cachehelper.NewRedisCache(
//	    cachehelper.WithRedisClient(cluster),
//	)
func WithRedisClient(client redis.UniversalClient) RedisOption {
	return cachesp.WithRedisClient(client)
}
