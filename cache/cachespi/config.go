package cachespi

import "time"

// Backend identifies the cache backend selected by CacheConfig.
type Backend string

const (
	// BackendInMemory stores cache data in the current process with Ristretto.
	BackendInMemory Backend = "in_memory"

	// BackendRedis stores cache data in Redis.
	BackendRedis Backend = "redis"
)

// CodecName selects a built-in cache codec from configuration.
type CodecName string

const (
	// CodecDefault uses the backend default codec.
	//
	// In-memory cache defaults to reference semantics and does not serialize
	// values. Redis defaults to JSON because it stores bytes.
	CodecDefault CodecName = ""

	// CodecJSON serializes values with encoding/json.
	CodecJSON CodecName = "json"
)

// CacheConfig is the user-facing configuration used by cachehelper.NewCache.
//
// Example YAML:
//
//	cache:
//	  backend: redis
//	  redis:
//	    addr: 127.0.0.1:6379
//	    default_ttl: 10m
//	    codec: json
type CacheConfig struct {
	Backend  Backend        `json:"backend" yaml:"backend"`
	InMemory InMemoryConfig `json:"in_memory" yaml:"in_memory"`
	Redis    RedisConfig    `json:"redis" yaml:"redis"`
}

// InMemoryConfig configures the Ristretto-backed in-memory cache.
//
// Zero values use production-safe defaults:
//   - default_ttl: 5 minutes
//   - num_counters: 1e7
//   - max_cost: 1 GiB
//   - buffer_items: 64
//   - codec: empty, meaning reference semantics
type InMemoryConfig struct {
	DefaultTTL  time.Duration `json:"default_ttl" yaml:"default_ttl"`
	NumCounters int64         `json:"num_counters" yaml:"num_counters"`
	MaxCost     int64         `json:"max_cost" yaml:"max_cost"`
	BufferItems int64         `json:"buffer_items" yaml:"buffer_items"`
	Codec       CodecName     `json:"codec" yaml:"codec"`
}

// RedisConfig configures the Redis-backed cache.
//
// Zero values use these defaults:
//   - addr: localhost:6379
//   - db: 0
//   - pool_size: 10
//   - min_idle_conns: 2
//   - dial_timeout: 5 seconds
//   - read_timeout: 3 seconds
//   - write_timeout: 3 seconds
//   - default_ttl: 5 minutes
//   - codec: json
type RedisConfig struct {
	Addr         string        `json:"addr" yaml:"addr"`
	Password     string        `json:"password" yaml:"password"`
	DB           int           `json:"db" yaml:"db"`
	PoolSize     int           `json:"pool_size" yaml:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns" yaml:"min_idle_conns"`
	DialTimeout  time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`
	DefaultTTL   time.Duration `json:"default_ttl" yaml:"default_ttl"`
	Codec        CodecName     `json:"codec" yaml:"codec"`
}
