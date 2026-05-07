// Package cachespi defines the public cache interfaces and shared types.
//
// Cache implementations are created from cachehelper. Callers usually depend on
// the Cache interface from this package and keep backend settings in
// CacheConfig, InMemoryConfig, or RedisConfig.
package cachespi
