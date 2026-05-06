package cachesp

import (
	"context"
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
	codec        cachespi.Codec
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

// WithRedisCodec sets the byte serializer used by RedisCache.
func WithRedisCodec(codec cachespi.Codec) RedisOption {
	return func(c *redisConfig) { c.codec = codec }
}

// RedisCache implements cachespi.Cache using Redis as the backend.
// Values are serialized/deserialized using the configured codec.
type RedisCache struct {
	client     *redis.Client
	defaultTTL time.Duration
	codec      binaryValueCodec
}

// NewRedisCache creates a new Cache backed by Redis.
func NewRedisCache(opts ...RedisOption) (*RedisCache, error) {
	cfg := defaultRedisConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	if _, err := resolveTTL(cachespi.DefaultExpiration, cfg.defaultTTL); err != nil {
		return nil, err
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.addr,
		Password:     cfg.password,
		DB:           cfg.db,
		PoolSize:     cfg.poolSize,
		MinIdleConns: cfg.minIdleConns,
		DialTimeout:  cfg.dialTimeout,
		ReadTimeout:  cfg.readTimeout,
		WriteTimeout: cfg.writeTimeout,
	})

	return &RedisCache{
		client:     client,
		defaultTTL: cfg.defaultTTL,
		codec:      newBinaryValueCodec(cfg.codec),
	}, nil
}

func (c *RedisCache) Close(_ context.Context) error {
	return c.client.Close()
}

func (c *RedisCache) Get(ctx context.Context, key string, receiver any) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return cachespi.ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("cache: redis GET %q: %w", key, err)
	}
	return c.codec.decodeBytes(data, receiver)
}

func (c *RedisCache) GetOrDefault(ctx context.Context, key string, defaultVal any, receiver any) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return setReceiver(receiver, defaultVal)
	}
	if err != nil {
		return fmt.Errorf("cache: redis GET %q: %w", key, err)
	}
	return c.codec.decodeBytes(data, receiver)
}

func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("cache: redis EXISTS %q: %w", key, err)
	}
	return n > 0, nil
}

func (c *RedisCache) GetMany(ctx context.Context, receiverMap map[string]any) error {
	if len(receiverMap) == 0 {
		return nil
	}

	keys := make([]string, 0, len(receiverMap))
	for k := range receiverMap {
		keys = append(keys, k)
	}

	vals, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return fmt.Errorf("cache: redis MGET: %w", err)
	}

	var missingKeys []string
	for i, key := range keys {
		if vals[i] == nil {
			missingKeys = append(missingKeys, key)
			continue
		}
		data, dataErr := bytesFromStoredValue(vals[i])
		if dataErr != nil {
			return fmt.Errorf("cache: redis MGET %q: %w", key, dataErr)
		}
		if err := c.codec.decodeBytes(data, receiverMap[key]); err != nil {
			return fmt.Errorf("cache: redis MGET %q: %w", key, err)
		}
	}
	for _, key := range missingKeys {
		delete(receiverMap, key)
	}
	return nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value any, expire time.Duration) error {
	ttl, err := resolveTTL(expire, c.defaultTTL)
	if err != nil {
		return err
	}
	data, err := c.codec.encodeBytes(value)
	if err != nil {
		return fmt.Errorf("cache: failed to encode value for key %q: %w", key, err)
	}
	return c.client.Set(ctx, key, data, ttl).Err()
}

func (c *RedisCache) SetNX(ctx context.Context, key string, value any, expire time.Duration) (bool, error) {
	ttl, err := resolveTTL(expire, c.defaultTTL)
	if err != nil {
		return false, err
	}
	data, err := c.codec.encodeBytes(value)
	if err != nil {
		return false, fmt.Errorf("cache: failed to encode value for key %q: %w", key, err)
	}
	ok, err := c.client.SetNX(ctx, key, data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("cache: redis SETNX %q: %w", key, err)
	}
	return ok, nil
}

func (c *RedisCache) GetAndDelete(ctx context.Context, key string, receiver any) error {
	data, err := c.client.GetDel(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		return cachespi.ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("cache: redis GETDEL %q: %w", key, err)
	}
	return c.codec.decodeBytes(data, receiver)
}

func (c *RedisCache) SetMany(ctx context.Context, valueMap map[string]any, expire time.Duration) error {
	if len(valueMap) == 0 {
		return nil
	}
	dataMap := make(map[string][]byte, len(valueMap))
	for key, value := range valueMap {
		data, err := c.codec.encodeBytes(value)
		if err != nil {
			return fmt.Errorf("cache: failed to encode value for key %q: %w", key, err)
		}
		dataMap[key] = data
	}
	return c.setManyEncoded(ctx, dataMap, expire)
}

func (c *RedisCache) setManyEncoded(ctx context.Context, dataMap map[string][]byte, expire time.Duration) error {
	if len(dataMap) == 0 {
		return nil
	}
	ttl, err := resolveTTL(expire, c.defaultTTL)
	if err != nil {
		return err
	}
	pipe := c.client.Pipeline()
	for key, data := range dataMap {
		pipe.Set(ctx, key, data, ttl)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (c *RedisCache) Delete(ctx context.Context, key string) error {
	n, err := c.client.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("cache: redis DEL %q: %w", key, err)
	}
	if n == 0 {
		return cachespi.ErrCacheMiss
	}
	return nil
}

func (c *RedisCache) DeleteMany(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

func (c *RedisCache) Load(ctx context.Context, loader cachespi.DataLoader, key string, receiver any,
	expire time.Duration) error {

	if err := c.Get(ctx, key, receiver); err == nil {
		return nil
	} else if !errors.Is(err, cachespi.ErrCacheMiss) {
		return err
	}
	ttl, err := resolveTTL(expire, c.defaultTTL)
	if err != nil {
		return err
	}
	if loader == nil {
		return fmt.Errorf("cache: loader must not be nil")
	}

	results, err := loader(ctx, []string{key})
	if err != nil {
		return err
	}
	if len(results) == 0 || results[0] == nil {
		return cachespi.ErrCacheMiss
	}

	data, err := c.codec.encodeBytes(results[0])
	if err != nil {
		return fmt.Errorf("cache: failed to encode value for key %q: %w", key, err)
	}
	if err := c.codec.decodeBytes(data, receiver); err != nil {
		return err
	}
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("cache: redis SET %q: %w", key, err)
	}
	return nil
}

func (c *RedisCache) LoadMany(ctx context.Context, loader cachespi.DataLoader, receiverMap map[string]any,
	expire time.Duration) error {

	if len(receiverMap) == 0 {
		return nil
	}

	var missingKeys []string

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
			missingKeys = append(missingKeys, key)
			continue
		}
		data, dataErr := bytesFromStoredValue(vals[i])
		if dataErr != nil {
			return fmt.Errorf("cache: redis MGET %q: %w", key, dataErr)
		}
		if jsonErr := c.codec.decodeBytes(data, receiverMap[key]); jsonErr != nil {
			return fmt.Errorf("cache: redis MGET %q: %w", key, jsonErr)
		}
	}

	if len(missingKeys) == 0 {
		return nil
	}
	if _, err := resolveTTL(expire, c.defaultTTL); err != nil {
		return err
	}
	if loader == nil {
		return fmt.Errorf("cache: loader must not be nil")
	}

	results, err := loader(ctx, missingKeys)
	if err != nil {
		return err
	}

	dataMap := make(map[string][]byte, len(missingKeys))
	var nilKeys []string
	for i, key := range missingKeys {
		if i >= len(results) || results[i] == nil {
			nilKeys = append(nilKeys, key)
			continue
		}

		data, encodeErr := c.codec.encodeBytes(results[i])
		if encodeErr != nil {
			return fmt.Errorf("cache: failed to encode value for key %q: %w", key, encodeErr)
		}
		if jsonErr := c.codec.decodeBytes(data, receiverMap[key]); jsonErr != nil {
			return fmt.Errorf("cache: key %q: %w", key, jsonErr)
		}
		dataMap[key] = data
	}
	if err := c.setManyEncoded(ctx, dataMap, expire); err != nil {
		return err
	}
	for _, key := range nilKeys {
		delete(receiverMap, key)
	}
	return nil
}

func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
