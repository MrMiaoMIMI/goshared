package cachespi

// Error defines cache error
type Error struct {
	Message string
}

// Error defines error string content
func (c Error) Error() string {
	return "cache:" + c.Message
}

func cacheErr(msg string) Error {
	return Error{Message: msg}
}

var (
	// ErrCacheMiss means that a Get failed because the item wasn't present.
	ErrCacheMiss = cacheErr("cache_miss")
)
