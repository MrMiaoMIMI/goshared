// Package cachehelper creates cache implementations backed by in-memory
// Ristretto or Redis stores.
//
// In-memory cache defaults to reference semantics: cached pointer, map, and
// slice values can be mutated through the original value or a returned receiver.
// Use WithInMemCodec, for example with cachespi.JSONCodec, when copy semantics
// are required.
//
// Redis cache stores serialized bytes and defaults to cachespi.JSONCodec.
// Reference semantics are not supported by Redis.
package cachehelper
