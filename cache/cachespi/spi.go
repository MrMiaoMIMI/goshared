package cachespi

import (
	"context"
	"time"
)

const (
	// NoExpiration makes a cached key never expire.
	NoExpiration time.Duration = -1

	// DefaultExpiration uses the default TTL configured on the cache instance.
	DefaultExpiration time.Duration = 0
)

// Cache is the common interface implemented by all cache backends.
//
// Expiration rules:
//   - NoExpiration keeps the key until it is explicitly deleted or evicted by
//     the backend.
//   - DefaultExpiration uses the cache-level default TTL. The helper
//     constructors default this value to 5 minutes unless an option overrides it.
//   - Any other negative duration is rejected with ErrInvalidExpiration.
//
// Receiver arguments must be non-nil pointers. Batch read methods may populate
// some receivers before returning an error; callers should only rely on batch
// results when the returned error is nil.
type Cache interface {
	// Get retrieves an item from the cache and populates receiver.
	// Returns ErrCacheMiss if the cache key does not exist.
	Get(ctx context.Context, key string, receiver any) error

	// GetOrDefault retrieves an item from the cache or populates receiver with
	// defaultVal when the key does not exist.
	GetOrDefault(ctx context.Context, key string, defaultVal any, receiver any) error

	// Exists checks whether the key exists in the cache.
	Exists(ctx context.Context, key string) (bool, error)

	// GetMany retrieves multiple items from the cache and populates receiverMap
	// values by key.
	//
	// On success, missing keys are removed from receiverMap.
	GetMany(ctx context.Context, receiverMap map[string]any) error

	// Set sets an item to the cache, replacing any existing item.
	// If expire is DefaultExpiration, it will use default expiration of the cache.
	Set(ctx context.Context, key string, value any, expire time.Duration) error

	// SetNX sets an item only if the key does not already exist.
	// Returns true if the key was set, false if it already existed.
	//
	// In-memory SetNX is atomic within one process. Redis SetNX is atomic across
	// clients connected to the same Redis deployment.
	SetNX(ctx context.Context, key string, value any, expire time.Duration) (bool, error)

	// GetAndDelete retrieves an item and removes it from the cache atomically.
	// Returns ErrCacheMiss if the key does not exist.
	GetAndDelete(ctx context.Context, key string, receiver any) error

	// SetMany sets multiple items to the cache, replacing any existing items.
	// If expire is DefaultExpiration, it will use the default expiration of the cache.
	SetMany(ctx context.Context, valueMap map[string]any, expire time.Duration) error

	// Delete removes an item from the cache.
	// Returns ErrCacheMiss if the cache key does not exist.
	Delete(ctx context.Context, key string) error

	// DeleteMany deletes multiple items from the cache. Missing keys are ignored.
	DeleteMany(ctx context.Context, keys []string) error

	// Load is similar to Get. If the key does not exist, it invokes loader,
	// stores the loaded value in the cache, and populates receiver.
	//
	// A nil or missing loader result is treated as ErrCacheMiss and is not stored.
	// If expire is DefaultExpiration, it will use the default expiration of the cache.
	Load(ctx context.Context, loader DataLoader, key string, receiver any, expire time.Duration) error

	// LoadMany is similar to GetMany. If some keys do not exist, it invokes
	// loader for the missing keys, stores loaded values in the cache, and
	// populates receiverMap.
	//
	// On success, cache misses and nil loader results are removed from
	// receiverMap.
	// If expire is DefaultExpiration, it will use the default expiration of the cache.
	LoadMany(ctx context.Context, loader DataLoader, receiverMap map[string]any, expire time.Duration) error

	// Ping checks the accessibility to the cache.
	Ping(ctx context.Context) error

	// Close releases cache resources. After Close returns, the cache should not
	// be reused.
	Close(ctx context.Context) error
}

// DataLoader loads values for the requested keys from a downstream service.
//
// The returned slice must align with keys by index. A nil result means that the
// corresponding key is not found and will not be stored.
//
// Loader implementations should honor ctx deadlines/cancellation to prevent
// hanging cache requests.
type DataLoader func(ctx context.Context, keys []string) ([]any, error)
