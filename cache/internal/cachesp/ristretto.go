package cachesp

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/dgraph-io/ristretto/v2"
)

var _ cachespi.Cache = (*RistrettoCache)(nil)

// RistrettoOption configures the RistrettoCache.
type RistrettoOption func(*cacheConfig)

type cacheConfig struct {
	numCounters int64
	maxCost     int64
	bufferItems int64
	defaultTTL  time.Duration
	codec       cachespi.Codec
}

func defaultConfig() *cacheConfig {
	return &cacheConfig{
		numCounters: 1e7,
		maxCost:     1 << 30,
		bufferItems: 64,
		defaultTTL:  5 * time.Minute,
	}
}

func WithRistrettoDefaultTTL(d time.Duration) RistrettoOption {
	return func(c *cacheConfig) { c.defaultTTL = d }
}

func WithRistrettoNumCounters(n int64) RistrettoOption {
	return func(c *cacheConfig) { c.numCounters = n }
}

func WithRistrettoMaxCost(n int64) RistrettoOption {
	return func(c *cacheConfig) { c.maxCost = n }
}

func WithRistrettoBufferItems(n int64) RistrettoOption {
	return func(c *cacheConfig) { c.bufferItems = n }
}

func WithRistrettoCodec(codec cachespi.Codec) RistrettoOption {
	return func(c *cacheConfig) { c.codec = codec }
}

// RistrettoCache implements cachespi.Cache using dgraph-io/ristretto as the backend.
type RistrettoCache struct {
	cache      *ristretto.Cache[string, any]
	defaultTTL time.Duration
	codec      valueCodec
	mu         sync.Mutex
	closed     bool
}

func NewRistrettoCache(opts ...RistrettoOption) (*RistrettoCache, error) {
	cfg := defaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	if _, err := resolveTTL(cachespi.DefaultExpiration, cfg.defaultTTL); err != nil {
		return nil, err
	}

	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: cfg.numCounters,
		MaxCost:     cfg.maxCost,
		BufferItems: cfg.bufferItems,
	})
	if err != nil {
		return nil, fmt.Errorf("cache: failed to create ristretto cache: %w", err)
	}

	codec := valueCodec(referenceValueCodec{})
	if cfg.codec != nil {
		codec = newBinaryValueCodec(cfg.codec)
	}

	return &RistrettoCache{
		cache:      cache,
		defaultTTL: cfg.defaultTTL,
		codec:      codec,
	}, nil
}

func (c *RistrettoCache) Close(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.cache.Close()
	c.closed = true
	return nil
}

func (c *RistrettoCache) Get(_ context.Context, key string, receiver any) error {
	val, found := c.cache.Get(key)
	if !found {
		return cachespi.ErrCacheMiss
	}
	return c.codec.decode(val, receiver)
}

func (c *RistrettoCache) GetOrDefault(_ context.Context, key string, defaultVal any, receiver any) error {
	val, found := c.cache.Get(key)
	if !found {
		return setReceiver(receiver, defaultVal)
	}
	return c.codec.decode(val, receiver)
}

func (c *RistrettoCache) Exists(_ context.Context, key string) (bool, error) {
	_, found := c.cache.Get(key)
	return found, nil
}

func (c *RistrettoCache) GetMany(_ context.Context, receiverMap map[string]any) error {
	var missingKeys []string
	for key, receiver := range receiverMap {
		val, found := c.cache.Get(key)
		if !found {
			missingKeys = append(missingKeys, key)
			continue
		}
		if err := c.codec.decode(val, receiver); err != nil {
			return fmt.Errorf("cache: key %q: %w", key, err)
		}
	}
	for _, key := range missingKeys {
		delete(receiverMap, key)
	}
	return nil
}

func (c *RistrettoCache) Set(_ context.Context, key string, value any, expire time.Duration) error {
	encoded, err := c.codec.encode(value)
	if err != nil {
		return fmt.Errorf("cache: failed to encode value for key %q: %w", key, err)
	}
	return c.setManyEncoded(map[string]any{key: encoded}, expire)
}

func (c *RistrettoCache) SetNX(_ context.Context, key string, value any, expire time.Duration) (bool, error) {
	ttl, err := resolveTTL(expire, c.defaultTTL)
	if err != nil {
		return false, err
	}
	encoded, err := c.codec.encode(value)
	if err != nil {
		return false, fmt.Errorf("cache: failed to encode value for key %q: %w", key, err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	_, found := c.cache.Get(key)
	if found {
		return false, nil
	}
	if ok := c.cache.SetWithTTL(key, encoded, 1, ttl); !ok {
		return false, fmt.Errorf("%w: key %q", cachespi.ErrCacheSetDropped, key)
	}
	c.cache.Wait()
	return true, nil
}

func (c *RistrettoCache) GetAndDelete(_ context.Context, key string, receiver any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	val, found := c.cache.Get(key)
	if !found {
		return cachespi.ErrCacheMiss
	}
	c.cache.Del(key)
	return c.codec.decode(val, receiver)
}

func (c *RistrettoCache) SetMany(_ context.Context, valueMap map[string]any, expire time.Duration) error {
	if len(valueMap) == 0 {
		return nil
	}

	encodedMap := make(map[string]any, len(valueMap))
	for key, value := range valueMap {
		encoded, err := c.codec.encode(value)
		if err != nil {
			return fmt.Errorf("cache: failed to encode value for key %q: %w", key, err)
		}
		encodedMap[key] = encoded
	}
	return c.setManyEncoded(encodedMap, expire)
}

func (c *RistrettoCache) setManyEncoded(valueMap map[string]any, expire time.Duration) error {
	if len(valueMap) == 0 {
		return nil
	}

	ttl, err := resolveTTL(expire, c.defaultTTL)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	var firstErr error
	for key, value := range valueMap {
		if ok := c.cache.SetWithTTL(key, value, 1, ttl); !ok && firstErr == nil {
			firstErr = fmt.Errorf("%w: key %q", cachespi.ErrCacheSetDropped, key)
		}
	}
	c.cache.Wait()
	return firstErr
}

func (c *RistrettoCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, found := c.cache.Get(key)
	if !found {
		return cachespi.ErrCacheMiss
	}
	// Del() removes the item from storedItems immediately; no Wait() needed.
	c.cache.Del(key)
	return nil
}

func (c *RistrettoCache) DeleteMany(_ context.Context, keys []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range keys {
		c.cache.Del(key)
	}
	return nil
}

func (c *RistrettoCache) Load(ctx context.Context, loader cachespi.DataLoader, key string, receiver any,
	expire time.Duration) error {

	if err := c.Get(ctx, key, receiver); err == nil {
		return nil
	} else if !errors.Is(err, cachespi.ErrCacheMiss) {
		return err
	}
	if _, err := resolveTTL(expire, c.defaultTTL); err != nil {
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

	encoded, err := c.codec.encode(results[0])
	if err != nil {
		return fmt.Errorf("cache: failed to encode value for key %q: %w", key, err)
	}
	if err := c.codec.decode(encoded, receiver); err != nil {
		return err
	}
	return c.setManyEncoded(map[string]any{key: encoded}, expire)
}

func (c *RistrettoCache) LoadMany(ctx context.Context, loader cachespi.DataLoader, receiverMap map[string]any,
	expire time.Duration) error {

	if len(receiverMap) == 0 {
		return nil
	}

	var missingKeys []string

	for key, receiver := range receiverMap {
		val, found := c.cache.Get(key)
		if found {
			if err := c.codec.decode(val, receiver); err != nil {
				return fmt.Errorf("cache: key %q: %w", key, err)
			}
		} else {
			missingKeys = append(missingKeys, key)
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

	encodedMap := make(map[string]any, len(missingKeys))
	var nilKeys []string
	for i, key := range missingKeys {
		if i >= len(results) || results[i] == nil {
			nilKeys = append(nilKeys, key)
			continue
		}

		encoded, err := c.codec.encode(results[i])
		if err != nil {
			return fmt.Errorf("cache: failed to encode value for key %q: %w", key, err)
		}
		if err := c.codec.decode(encoded, receiverMap[key]); err != nil {
			return fmt.Errorf("cache: key %q: %w", key, err)
		}
		encodedMap[key] = encoded
	}
	if err := c.setManyEncoded(encodedMap, expire); err != nil {
		return err
	}
	for _, key := range nilKeys {
		delete(receiverMap, key)
	}
	return nil
}

func (c *RistrettoCache) Ping(_ context.Context) error {
	return nil
}
