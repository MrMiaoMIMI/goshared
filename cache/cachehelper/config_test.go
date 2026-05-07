package cachehelper

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/alicebob/miniredis/v2"
)

type product struct {
	ID   int64
	Name string
}

func TestNewCacheRequiresBackend(t *testing.T) {
	_, err := NewCache(cachespi.CacheConfig{})
	if err == nil {
		t.Fatalf("expected backend error")
	}
}

func TestNewCacheCreatesInMemoryBackendFromConfig(t *testing.T) {
	ctx := context.Background()
	cache, err := NewCache(cachespi.CacheConfig{
		Backend: cachespi.BackendInMemory,
		InMemory: cachespi.InMemoryConfig{
			DefaultTTL: 5 * time.Minute,
			Codec:      cachespi.CodecJSON,
		},
	})
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	t.Cleanup(func() {
		if err := cache.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	source := &product{ID: 1, Name: "original"}
	if err := cache.Set(ctx, "product:1", source, cachespi.DefaultExpiration); err != nil {
		t.Fatalf("Set: %v", err)
	}
	source.Name = "mutated"

	var got product
	if err := cache.Get(ctx, "product:1", &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "original" {
		t.Fatalf("expected JSON codec copy semantics, got %#v", got)
	}
}

func TestNewCacheCreatesRedisBackendFromConfig(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	cache, err := NewCache(cachespi.CacheConfig{
		Backend: cachespi.BackendRedis,
		Redis: cachespi.RedisConfig{
			Addr:       server.Addr(),
			DefaultTTL: 5 * time.Minute,
			Codec:      cachespi.CodecJSON,
		},
	})
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}
	t.Cleanup(func() {
		if err := cache.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	if err := cache.Set(ctx, "greeting", "hello", cachespi.DefaultExpiration); err != nil {
		t.Fatalf("Set: %v", err)
	}
	var got string
	if err := cache.Get(ctx, "greeting", &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "hello" {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestNewInMemCacheRejectsUnsupportedCodec(t *testing.T) {
	_, err := NewInMemCache(cachespi.InMemoryConfig{Codec: "msgpack"})
	if err == nil {
		t.Fatalf("expected unsupported codec error")
	}
}

func TestNewInMemCacheRejectsInvalidDefaultTTL(t *testing.T) {
	_, err := NewInMemCache(cachespi.InMemoryConfig{DefaultTTL: -2 * time.Second})
	if !errors.Is(err, cachespi.ErrInvalidExpiration) {
		t.Fatalf("expected ErrInvalidExpiration, got %v", err)
	}
}
