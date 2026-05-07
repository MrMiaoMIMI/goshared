// Package cachehelper creates cache implementations backed by in-memory
// Ristretto or Redis stores.
//
// Prefer NewCache with cachespi.CacheConfig in business services so cache
// backend, connection, TTL, and codec settings can live in application config
// files. NewInMemCache and NewRedisCache are available for focused tests and
// small programs that already know the backend.
//
// In-memory cache defaults to reference semantics: cached pointer, map, and
// slice values can be mutated through the original value or a returned receiver.
// Set cachespi.InMemoryConfig.Codec to cachespi.CodecJSON when copy semantics
// are required.
//
// Redis cache stores serialized bytes and defaults to cachespi.JSONCodec.
// Reference semantics are not supported by Redis.
package cachehelper
