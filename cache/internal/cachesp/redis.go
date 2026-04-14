package cachesp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/redis/go-redis/v9"
)

var _ cachespi.Cache = (*RedisCache)(nil)

// RedisOption configures the RedisCache.
type RedisOption func(*redisConfig)

type redisConfig struct {
	addr         string
	password     string
	db           int
	poolSize     int
	minIdleConns int
	dialTimeout  time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	defaultTTL   time.Duration
	client       redis.UniversalClient // allow injecting a pre-built client
}

func defaultRedisConfig() *redisConfig {
	return &redisConfig{
		addr:         "localhost:6379",
		db:           0,
		poolSize:     10,
		minIdleConns: 2,
		dialTimeout:  5 * time.Second,
		readTimeout:  3 * time.Second,
		writeTimeout: 3 * time.Second,
		defaultTTL:   5 * time.Minute,
	}
}

// WithRedisAddr sets the Redis server address (host:port).
func WithRedisAddr(addr string) RedisOption {
	return func(c *redisConfig) { c.addr = addr }
}

// WithRedisPassword sets the Redis password.
func WithRedisPassword(password string) RedisOption {
	return func(c *redisConfig) { c.password = password }
}

// WithRedisDB selects the Redis database index.
func WithRedisDB(db int) RedisOption {
	return func(c *redisConfig) { c.db = db }
}

// WithRedisPoolSize sets the maximum number of connections in the pool.
func WithRedisPoolSize(size int) RedisOption {
	return func(c *redisConfig) { c.poolSize = size }
}

// WithRedisMinIdleConns sets the minimum number of idle connections.
func WithRedisMinIdleConns(n int) RedisOption {
	return func(c *redisConfig) { c.minIdleConns = n }
}

// WithRedisDialTimeout sets the timeout for establishing new connections.
func WithRedisDialTimeout(d time.Duration) RedisOption {
	return func(c *redisConfig) { c.dialTimeout = d }
}

// WithRedisReadTimeout sets the timeout for socket reads.
func WithRedisReadTimeout(d time.Duration) RedisOption {
	return func(c *redisConfig) { c.readTimeout = d }
}

// WithRedisWriteTimeout sets the timeout for socket writes.
func WithRedisWriteTimeout(d time.Duration) RedisOption {
	return func(c *redisConfig) { c.writeTimeout = d }
}

// WithRedisDefaultTTL sets the default TTL for cache entries.
func WithRedisDefaultTTL(d time.Duration) RedisOption {
	return func(c *redisConfig) { c.defaultTTL = d }
}

// WithRedisClient allows injecting a pre-configured redis.UniversalClient.
// When set, all connection options (addr, password, etc.) are ignored.
func WithRedisClient(client redis.UniversalClient) RedisOption {
	return func(c *redisConfig) { c.client = client }
}

// RedisCache implements cachespi.Cache using Redis as the backend.
// Values are serialized/deserialized using JSON.
type RedisCache struct {
	client     redis.UniversalClient
	defaultTTL time.Duration
}

// NewRedisCache creates a new Cache backed by Redis.
func NewRedisCache(opts ...RedisOption) cachespi.Cache {
	cfg := defaultRedisConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	client := cfg.client
	if client == nil {
		client = redis.NewClient(&redis.Options{
			Addr:         cfg.addr,
			Password:     cfg.password,
			DB:           cfg.db,
			PoolSize:     cfg.poolSize,
			MinIdleConns: cfg.minIdleConns,
			DialTimeout:  cfg.dialTimeout,
			ReadTimeout:  cfg.readTimeout,
			WriteTimeout: cfg.writeTimeout,
		})
	}

	return &RedisCache{
		client:     client,
		defaultTTL: cfg.defaultTTL,
	}
}

func (c *RedisCache) resolveTTL(expire time.Duration) time.Duration {
	switch expire {
	case cachespi.NoExpiration:
		return 0
	case cachespi.DefaultExpiration:
		return c.defaultTTL
	default:
		return expire
	}
}

func (c *RedisCache) Get(ctx context.Context, key string, receiver any, _ ...cachespi.OperationOption) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return cachespi.ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("cache: redis GET %q: %w", key, err)
	}
	return json.Unmarshal(data, receiver)
}

func (c *RedisCache) GetOrDefault(ctx context.Context, key string, defaultVal any, receiver any, _ ...cachespi.OperationOption) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return setReceiver(receiver, defaultVal)
	}
	if err != nil {
		return fmt.Errorf("cache: redis GET %q: %w", key, err)
	}
	return json.Unmarshal(data, receiver)
}

func (c *RedisCache) Exists(ctx context.Context, key string, _ ...cachespi.OperationOption) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("cache: redis EXISTS %q: %w", key, err)
	}
	return n > 0, nil
}

func (c *RedisCache) GetMany(ctx context.Context, receiverMap map[string]any, _ ...cachespi.OperationOption) error {
	keys := make([]string, 0, len(receiverMap))
	for k := range receiverMap {
		keys = append(keys, k)
	}

	vals, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return fmt.Errorf("cache: redis MGET: %w", err)
	}

	for i, key := range keys {
		if vals[i] == nil {
			delete(receiverMap, key)
			continue
		}
		str, ok := vals[i].(string)
		if !ok {
			delete(receiverMap, key)
			continue
		}
		if err := json.Unmarshal([]byte(str), receiverMap[key]); err != nil {
			delete(receiverMap, key)
		}
	}
	return nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value any, expire time.Duration, _ ...cachespi.OperationOption) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache: failed to marshal value for key %q: %w", key, err)
	}
	ttl := c.resolveTTL(expire)
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisCache) SetNX(ctx context.Context, key string, value any, expire time.Duration, _ ...cachespi.OperationOption) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("cache: failed to marshal value for key %q: %w", key, err)
	}
	ttl := c.resolveTTL(expire)
	ok, err := c.client.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("cache: redis SETNX %q: %w", key, err)
	}
	return ok, nil
}

func (c *RedisCache) GetAndDelete(ctx context.Context, key string, receiver any, _ ...cachespi.OperationOption) error {
	data, err := c.client.GetDel(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return cachespi.ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("cache: redis GETDEL %q: %w", key, err)
	}
	return json.Unmarshal(data, receiver)
}

func (c *RedisCache) SetMany(ctx context.Context, valueMap map[string]any, expire time.Duration, _ ...cachespi.OperationOption) error {
	ttl := c.resolveTTL(expire)
	pipe := c.client.Pipeline()
	for key, value := range valueMap {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("cache: failed to marshal value for key %q: %w", key, err)
		}
		pipe.Set(ctx, key, data, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisCache) Delete(ctx context.Context, key string, _ ...cachespi.OperationOption) error {
	n, err := c.client.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("cache: redis DEL %q: %w", key, err)
	}
	if n == 0 {
		return cachespi.ErrCacheMiss
	}
	return nil
}

func (c *RedisCache) DeleteMany(ctx context.Context, keys []string, _ ...cachespi.OperationOption) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

func (c *RedisCache) Load(ctx context.Context, loader cachespi.DataLoader, key string, receiver any,
	expire time.Duration, _ ...cachespi.OperationOption) error {

	if err := c.Get(ctx, key, receiver); err == nil {
		return nil
	}

	results, err := loader(ctx, []string{key})
	if err != nil {
		return err
	}
	if len(results) == 0 || results[0] == nil {
		return cachespi.ErrCacheMiss
	}

	_ = c.Set(ctx, key, results[0], expire)

	data, err := json.Marshal(results[0])
	if err != nil {
		return err
	}
	return json.Unmarshal(data, receiver)
}

func (c *RedisCache) LoadMany(ctx context.Context, loader cachespi.DataLoader, receiverMap map[string]any,
	expire time.Duration, _ ...cachespi.OperationOption) error {

	var missingKeys []string

	keys := make([]string, 0, len(receiverMap))
	for k := range receiverMap {
		keys = append(keys, k)
	}

	vals, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		missingKeys = keys
	} else {
		for i, key := range keys {
			if vals[i] == nil {
				missingKeys = append(missingKeys, key)
				continue
			}
			str, ok := vals[i].(string)
			if !ok {
				missingKeys = append(missingKeys, key)
				continue
			}
			if jsonErr := json.Unmarshal([]byte(str), receiverMap[key]); jsonErr != nil {
				missingKeys = append(missingKeys, key)
			}
		}
	}

	if len(missingKeys) == 0 {
		return nil
	}

	results, err := loader(ctx, missingKeys)
	if err != nil {
		for _, key := range missingKeys {
			delete(receiverMap, key)
		}
		return err
	}

	for i, key := range missingKeys {
		if i >= len(results) || results[i] == nil {
			delete(receiverMap, key)
			continue
		}

		_ = c.Set(ctx, key, results[i], expire)

		data, marshalErr := json.Marshal(results[i])
		if marshalErr != nil {
			delete(receiverMap, key)
			continue
		}
		if jsonErr := json.Unmarshal(data, receiverMap[key]); jsonErr != nil {
			delete(receiverMap, key)
		}
	}
	return nil
}

func (c *RedisCache) Incr(ctx context.Context, key string, delta int64, expire time.Duration, _ ...cachespi.OperationOption) (int64, error) {
	newVal, err := c.client.IncrBy(ctx, key, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("cache: redis INCRBY %q: %w", key, err)
	}

	ttl := c.resolveTTL(expire)
	if ttl > 0 {
		if expErr := c.client.Expire(ctx, key, ttl).Err(); expErr != nil {
			return newVal, fmt.Errorf("cache: redis EXPIRE %q: %w", key, expErr)
		}
	}

	return newVal, nil
}

func (c *RedisCache) Flush(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
