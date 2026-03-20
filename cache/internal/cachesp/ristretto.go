package cachesp

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/dgraph-io/ristretto/v2"
)

var _ cachespi.Cache = (*RistrettoCache)(nil)

// CacheOption configures the RistrettoCache.
type CacheOption func(*cacheConfig)

type cacheConfig struct {
	numCounters int64
	maxCost     int64
	bufferItems int64
	defaultTTL  time.Duration
}

func defaultConfig() *cacheConfig {
	return &cacheConfig{
		numCounters: 1e7,
		maxCost:     1 << 30,
		bufferItems: 64,
		defaultTTL:  5 * time.Minute,
	}
}

func WithDefaultTTL(d time.Duration) CacheOption {
	return func(c *cacheConfig) { c.defaultTTL = d }
}

// RistrettoCache implements cachespi.Cache using dgraph-io/ristretto as the backend.
type RistrettoCache struct {
	cache      *ristretto.Cache[string, any]
	defaultTTL time.Duration
}

func NewRistrettoCache(opts ...CacheOption) cachespi.Cache {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	cache, err := ristretto.NewCache(&ristretto.Config[string, any]{
		NumCounters: cfg.numCounters,
		MaxCost:     cfg.maxCost,
		BufferItems: cfg.bufferItems,
	})
	if err != nil {
		panic(fmt.Sprintf("cache: failed to create ristretto cache: %v", err))
	}

	return &RistrettoCache{
		cache:      cache,
		defaultTTL: cfg.defaultTTL,
	}
}

// resolveTTL maps cachespi expiration semantics to ristretto TTL.
//   - NoExpiration (-1)      → 0 (ristretto: never expire)
//   - DefaultExpiration (0)  → configured defaultTTL
//   - positive duration      → used as-is
func (c *RistrettoCache) resolveTTL(expire time.Duration) time.Duration {
	switch expire {
	case cachespi.NoExpiration:
		return 0
	case cachespi.DefaultExpiration:
		return c.defaultTTL
	default:
		return expire
	}
}

func (c *RistrettoCache) Get(_ context.Context, key string, receiver any, _ ...cachespi.OperationOption) error {
	val, found := c.cache.Get(key)
	if !found {
		return cachespi.ErrCacheMiss
	}
	return setReceiver(receiver, val)
}

func (c *RistrettoCache) GetMany(_ context.Context, receiverMap map[string]any, _ ...cachespi.OperationOption) error {
	for key, receiver := range receiverMap {
		val, found := c.cache.Get(key)
		if !found {
			delete(receiverMap, key)
			continue
		}
		if err := setReceiver(receiver, val); err != nil {
			delete(receiverMap, key)
		}
	}
	return nil
}

func (c *RistrettoCache) Set(_ context.Context, key string, value any, expire time.Duration, _ ...cachespi.OperationOption) error {
	ttl := c.resolveTTL(expire)
	c.cache.SetWithTTL(key, value, 1, ttl)
	c.cache.Wait()
	return nil
}

func (c *RistrettoCache) SetMany(_ context.Context, valueMap map[string]any, expire time.Duration, _ ...cachespi.OperationOption) error {
	ttl := c.resolveTTL(expire)
	for key, value := range valueMap {
		c.cache.SetWithTTL(key, value, 1, ttl)
	}
	c.cache.Wait()
	return nil
}

func (c *RistrettoCache) Delete(_ context.Context, key string, _ ...cachespi.OperationOption) error {
	_, found := c.cache.Get(key)
	if !found {
		return cachespi.ErrCacheMiss
	}
	// Del() removes the item from storedItems immediately; no Wait() needed.
	c.cache.Del(key)
	return nil
}

func (c *RistrettoCache) DeleteMany(_ context.Context, keys []string, _ ...cachespi.OperationOption) error {
	for _, key := range keys {
		c.cache.Del(key)
	}
	return nil
}

func (c *RistrettoCache) Load(ctx context.Context, loader cachespi.DataLoader, key string, receiver any,
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
	return setReceiver(receiver, results[0])
}

func (c *RistrettoCache) LoadMany(ctx context.Context, loader cachespi.DataLoader, receiverMap map[string]any,
	expire time.Duration, _ ...cachespi.OperationOption) error {

	var missingKeys []string
	missingReceivers := make(map[string]any)

	for key, receiver := range receiverMap {
		val, found := c.cache.Get(key)
		if found {
			if err := setReceiver(receiver, val); err != nil {
				delete(receiverMap, key)
			}
		} else {
			missingKeys = append(missingKeys, key)
			missingReceivers[key] = receiver
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

		if receiver, ok := missingReceivers[key]; ok {
			if err := setReceiver(receiver, results[i]); err != nil {
				delete(receiverMap, key)
			}
		}
	}
	return nil
}

func (c *RistrettoCache) Flush(_ context.Context) error {
	// Clear() is synchronous — it stops processItems, drains setBuf,
	// clears storedItems + policy, then restarts processItems.
	c.cache.Clear()
	return nil
}

func (c *RistrettoCache) Ping(_ context.Context) error {
	return nil
}

// setReceiver copies the cached value into the receiver pointer via reflection.
func setReceiver(receiver any, value any) error {
	if receiver == nil {
		return fmt.Errorf("cache: receiver must be a non-nil pointer")
	}
	rv := reflect.ValueOf(receiver)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("cache: receiver must be a non-nil pointer")
	}

	val := reflect.ValueOf(value)
	targetType := rv.Elem().Type()

	if val.Type().AssignableTo(targetType) {
		rv.Elem().Set(val)
		return nil
	}

	if val.Kind() == reflect.Ptr && !val.IsNil() && val.Elem().Type().AssignableTo(targetType) {
		rv.Elem().Set(val.Elem())
		return nil
	}

	return fmt.Errorf("cache: cannot assign %T to %s", value, targetType)
}
