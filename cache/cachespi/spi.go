package cachespi

import (
	"context"
	"time"
)

const (
	// NoExpiration will make cached key never expire
	NoExpiration time.Duration = -1

	// DefaultExpiration will use the default expiration value set at cache level while configuration
	DefaultExpiration time.Duration = 0
)

// Cache is the interface of a cache store
type Cache interface {
	// Get retrieves an item from the cache, and populate into the receiver.
	// Returns ErrCacheMiss if the cache key does not exist.
	Get(ctx context.Context, key string, receiver any, opts ...OperationOption) error

	// GetOrDefault retrieves an item from the cache.
	// Returns defaultVal if the key does not exist instead of ErrCacheMiss.
	GetOrDefault(ctx context.Context, key string, defaultVal any, receiver any, opts ...OperationOption) error

	// Exists checks whether the key exists in the cache.
	Exists(ctx context.Context, key string, opts ...OperationOption) (bool, error)

	// GetMany retrieves multiple items from the cache. Set to receiverMap based on key.
	// The receiverMap may have fewer elements than original, due to cache misses.
	GetMany(ctx context.Context, receiverMap map[string]any, opts ...OperationOption) error

	// Set sets an item to the cache, replacing any existing item.
	// If expire is DefaultExpiration, it will use default expiration of the cache.
	Set(ctx context.Context, key string, value any, expire time.Duration, opts ...OperationOption) error

	// SetNX sets an item only if the key does not already exist.
	// Returns true if the key was set, false if it already existed.
	SetNX(ctx context.Context, key string, value any, expire time.Duration, opts ...OperationOption) (bool, error)

	// GetAndDelete retrieves an item and removes it from the cache atomically.
	// Returns ErrCacheMiss if the key does not exist.
	GetAndDelete(ctx context.Context, key string, receiver any, opts ...OperationOption) error

	// SetMany sets multiple items to the cache, replacing any existing items.
	// If expire is DefaultExpiration, it will use the default expiration of the cache.
	SetMany(ctx context.Context, valueMap map[string]any, expire time.Duration, opts ...OperationOption) error

	// Delete removes an item from the cache.
	// Returns ErrCacheMiss if the cache key does not exist.
	Delete(ctx context.Context, key string, opts ...OperationOption) error

	// DeleteMany deletes multiple items from the cache.
	DeleteMany(ctx context.Context, keys []string, opts ...OperationOption) error

	// Load is similar like Get, but if the key doesn't exist, it will invoke loader to load the data and store to cache
	// If expire is DefaultExpiration, it will use the default expiration of the cache.
	Load(ctx context.Context, loader DataLoader, key string, receiver any,
		expire time.Duration, opts ...OperationOption) error

	// LoadMany is similar like GetMany, but if some keys don't exist,
	// it will invoke loader to load the missing data and store to cache.
	// The receiverMap may have fewer elements than original, due to some data failed to get from both cache and loader.
	// If expire is DefaultExpiration, it will use the default expiration of the cache.
	LoadMany(ctx context.Context, loader DataLoader, receiverMap map[string]any, expire time.Duration, opts ...OperationOption) error

	// Incr increments the integer value of key by delta and returns the new value.
	// If the key doesn't exist, it is initialized to 0 before incrementing.
	Incr(ctx context.Context, key string, delta int64, expire time.Duration, opts ...OperationOption) (int64, error)

	// Flush deletes all items from the cache.
	Flush(ctx context.Context) error

	// Ping checks the accessibility to the cache.
	Ping(ctx context.Context) error
}

// DataLoader will return data from downstream service.
// When return result is nil, will not set it to cache stores.
//
// Downstream service MUST set TIMEOUT mechanism for DataLoader to prevent hanging requests.
type DataLoader func(ctx context.Context, keys []string) ([]any, error)
