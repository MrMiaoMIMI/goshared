// Package ctxutil provides utilities for working with context.Context,
// including typed key-value storage and deadline helpers.
package ctxutil

import (
	"context"
	"time"
)

type contextKey string

// WithValue stores a typed value in context under a namespaced key.
// The key string is converted to an unexported type to avoid collisions.
func WithValue(ctx context.Context, key string, val any) context.Context {
	return context.WithValue(ctx, contextKey(key), val)
}

// Value retrieves a typed value from context.
// Returns the zero value of T and false if the key is absent or the type doesn't match.
func Value[T any](ctx context.Context, key string) (T, bool) {
	v, ok := ctx.Value(contextKey(key)).(T)
	return v, ok
}

// MustValue retrieves a typed value from context.
// Panics if the key is absent or the type doesn't match.
func MustValue[T any](ctx context.Context, key string) T {
	v, ok := ctx.Value(contextKey(key)).(T)
	if !ok {
		panic("ctxutil: missing or wrong type for key " + key)
	}
	return v
}

// ValueOr retrieves a typed value from context, returning defaultVal if absent.
func ValueOr[T any](ctx context.Context, key string, defaultVal T) T {
	v, ok := ctx.Value(contextKey(key)).(T)
	if !ok {
		return defaultVal
	}
	return v
}

// Deadline returns the remaining time until the context deadline.
// If no deadline is set, returns maxDuration.
func Deadline(ctx context.Context, maxDuration time.Duration) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return maxDuration
	}
	remaining := time.Until(deadline)
	if remaining < 0 {
		return 0
	}
	if remaining > maxDuration {
		return maxDuration
	}
	return remaining
}

// IsExpired returns true if the context has been cancelled or its deadline has passed.
func IsExpired(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// Merge creates a context that is cancelled when either ctx1 or ctx2 is cancelled.
// Values are looked up from ctx1 first, then ctx2.
func Merge(ctx1, ctx2 context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx1)
	go func() {
		select {
		case <-ctx2.Done():
			cancel()
		case <-ctx.Done():
		}
	}()
	return &mergedCtx{Context: ctx, secondary: ctx2}, cancel
}

type mergedCtx struct {
	context.Context
	secondary context.Context
}

func (m *mergedCtx) Value(key any) any {
	if v := m.Context.Value(key); v != nil {
		return v
	}
	return m.secondary.Value(key)
}
