package cachespi

// Error defines a cache sentinel error.
type Error struct {
	Message string
}

// Error returns the cache error message.
func (c Error) Error() string {
	return "cache:" + c.Message
}

func cacheErr(msg string) Error {
	return Error{Message: msg}
}

var (
	// ErrCacheMiss means that a requested cache key was not present.
	ErrCacheMiss = cacheErr("cache_miss")

	// ErrInvalidExpiration means that a cache operation received a negative
	// expiration other than NoExpiration.
	ErrInvalidExpiration = cacheErr("invalid_expiration")

	// ErrCacheSetDropped means that the underlying cache rejected a set
	// operation. This is most likely with in-memory caches using admission or
	// capacity policies.
	ErrCacheSetDropped = cacheErr("cache_set_dropped")
)
