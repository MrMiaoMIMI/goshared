package cachesp

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

type testProduct struct {
	ID   int64
	Name string
}

func TestRistrettoGetOrDefaultNilUsesZeroValue(t *testing.T) {
	ctx := context.Background()
	cache := newRistrettoTestCache(t)

	value := "preset"
	if err := cache.GetOrDefault(ctx, "missing:string", nil, &value); err != nil {
		t.Fatalf("GetOrDefault nil default: %v", err)
	}
	if value != "" {
		t.Fatalf("expected zero string, got %q", value)
	}

	ptr := &testProduct{ID: 1}
	if err := cache.GetOrDefault(ctx, "missing:ptr", nil, &ptr); err != nil {
		t.Fatalf("GetOrDefault nil pointer default: %v", err)
	}
	if ptr != nil {
		t.Fatalf("expected nil pointer, got %#v", ptr)
	}
}

func TestRistrettoGetStructIntoPointerReceiver(t *testing.T) {
	ctx := context.Background()
	cache := newRistrettoTestCache(t)

	if err := cache.Set(ctx, "product", testProduct{ID: 7, Name: "widget"}, cachespi.NoExpiration); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var got *testProduct
	if err := cache.Get(ctx, "product", &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil || got.ID != 7 || got.Name != "widget" {
		t.Fatalf("unexpected product: %#v", got)
	}
}

func TestRistrettoRejectsInvalidExpiration(t *testing.T) {
	ctx := context.Background()
	cache := newRistrettoTestCache(t)

	err := cache.Set(ctx, "invalid-ttl", "value", -2*time.Second)
	if !errors.Is(err, cachespi.ErrInvalidExpiration) {
		t.Fatalf("expected ErrInvalidExpiration, got %v", err)
	}
}

func TestRistrettoLoadDoesNotFallbackOnReceiverError(t *testing.T) {
	ctx := context.Background()
	cache := newRistrettoTestCache(t)

	if err := cache.Set(ctx, "type-mismatch", "not-an-int", cachespi.NoExpiration); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var loaderCalls int32
	loader := func(context.Context, []string) ([]any, error) {
		atomic.AddInt32(&loaderCalls, 1)
		return []any{123}, nil
	}

	var got int
	err := cache.Load(ctx, loader, "type-mismatch", &got, cachespi.NoExpiration)
	if err == nil {
		t.Fatalf("expected receiver error")
	}
	if atomic.LoadInt32(&loaderCalls) != 0 {
		t.Fatalf("loader should not be called for non-miss cache errors")
	}
}

func TestRistrettoConcurrentSetNX(t *testing.T) {
	ctx := context.Background()
	cache := newRistrettoTestCache(t)

	const workers = 64
	start := make(chan struct{})
	var wg sync.WaitGroup
	var successes int32
	errCh := make(chan error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ok, err := cache.SetNX(ctx, "once", "value", cachespi.NoExpiration)
			if err != nil {
				errCh <- err
				return
			}
			if ok {
				atomic.AddInt32(&successes, 1)
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("SetNX failed: %v", err)
	}
	if successes != 1 {
		t.Fatalf("expected exactly one SetNX success, got %d", successes)
	}
}

func TestRistrettoDefaultCodecStoresReferences(t *testing.T) {
	ctx := context.Background()
	cache := newRistrettoTestCache(t)

	product := &testProduct{ID: 1, Name: "original"}
	if err := cache.Set(ctx, "product", product, cachespi.NoExpiration); err != nil {
		t.Fatalf("Set: %v", err)
	}

	product.Name = "mutated-source"
	var got *testProduct
	if err := cache.Get(ctx, "product", &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "mutated-source" {
		t.Fatalf("expected reference semantics, got %#v", got)
	}

	got.Name = "mutated-receiver"
	var gotAgain *testProduct
	if err := cache.Get(ctx, "product", &gotAgain); err != nil {
		t.Fatalf("Get again: %v", err)
	}
	if gotAgain.Name != "mutated-receiver" {
		t.Fatalf("expected receiver mutation to affect cached reference, got %#v", gotAgain)
	}
}

func TestRistrettoJSONCodecCopiesValues(t *testing.T) {
	ctx := context.Background()
	cache, err := NewRistrettoCache(RistrettoConfig{Codec: cachespi.JSONCodec{}})
	if err != nil {
		t.Fatalf("NewRistrettoCache: %v", err)
	}
	t.Cleanup(func() {
		if err := cache.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	product := &testProduct{ID: 1, Name: "original"}
	if err := cache.Set(ctx, "product", product, cachespi.NoExpiration); err != nil {
		t.Fatalf("Set: %v", err)
	}

	product.Name = "mutated-source"
	var got testProduct
	if err := cache.Get(ctx, "product", &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Name != "original" {
		t.Fatalf("expected copied cached value, got %#v", got)
	}

	got.Name = "mutated-receiver"
	var gotAgain testProduct
	if err := cache.Get(ctx, "product", &gotAgain); err != nil {
		t.Fatalf("Get again: %v", err)
	}
	if gotAgain.Name != "original" {
		t.Fatalf("expected receiver mutation not to affect cache, got %#v", gotAgain)
	}
}

func TestRedisRejectsInvalidExpiration(t *testing.T) {
	ctx := context.Background()
	cache, client, _ := newRedisTestCache(t)

	err := cache.Set(ctx, "invalid-ttl", "value", -2*time.Second)
	if !errors.Is(err, cachespi.ErrInvalidExpiration) {
		t.Fatalf("expected ErrInvalidExpiration, got %v", err)
	}

	exists, err := client.Exists(ctx, "invalid-ttl").Result()
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists != 0 {
		t.Fatalf("invalid ttl key should not be written")
	}
}

func TestRedisLoadDoesNotFallbackOnGetError(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	cache, err := NewRedisCache(RedisConfig{
		Addr:         server.Addr(),
		DialTimeout:  20 * time.Millisecond,
		ReadTimeout:  20 * time.Millisecond,
		WriteTimeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewRedisCache: %v", err)
	}
	t.Cleanup(func() {
		if err := cache.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	server.Close()

	var loaderCalls int32
	loader := func(context.Context, []string) ([]any, error) {
		atomic.AddInt32(&loaderCalls, 1)
		return []any{"loaded"}, nil
	}

	var got string
	err = cache.Load(ctx, loader, "key", &got, cachespi.NoExpiration)
	if err == nil || errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected redis connection error, got %v", err)
	}
	if atomic.LoadInt32(&loaderCalls) != 0 {
		t.Fatalf("loader should not be called for redis errors")
	}
}

func TestRedisLoadManyDoesNotFallbackOnMGetError(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	cache, err := NewRedisCache(RedisConfig{
		Addr:         server.Addr(),
		DialTimeout:  20 * time.Millisecond,
		ReadTimeout:  20 * time.Millisecond,
		WriteTimeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewRedisCache: %v", err)
	}
	t.Cleanup(func() {
		if err := cache.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})

	server.Close()

	var loaderCalls int32
	loader := func(context.Context, []string) ([]any, error) {
		atomic.AddInt32(&loaderCalls, 1)
		return []any{"loaded"}, nil
	}

	var got string
	receiverMap := map[string]any{"key": &got}
	err = cache.LoadMany(ctx, loader, receiverMap, cachespi.NoExpiration)
	if err == nil {
		t.Fatalf("expected redis connection error")
	}
	if atomic.LoadInt32(&loaderCalls) != 0 {
		t.Fatalf("loader should not be called for redis errors")
	}
}

func TestRedisGetManyReturnsDecodeError(t *testing.T) {
	ctx := context.Background()
	cache, client, _ := newRedisTestCache(t)

	if err := client.Set(ctx, "bad-json", "not-json", 0).Err(); err != nil {
		t.Fatalf("raw Set: %v", err)
	}

	var got string
	err := cache.GetMany(ctx, map[string]any{"bad-json": &got})
	if err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestRedisLoadDoesNotWriteWhenReceiverDecodeFails(t *testing.T) {
	ctx := context.Background()
	cache, client, _ := newRedisTestCache(t)

	loader := func(context.Context, []string) ([]any, error) {
		return []any{"not-an-int"}, nil
	}

	var got int
	err := cache.Load(ctx, loader, "bad-load", &got, cachespi.NoExpiration)
	if err == nil {
		t.Fatalf("expected decode error")
	}

	exists, err := client.Exists(ctx, "bad-load").Result()
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists != 0 {
		t.Fatalf("Load should not write cache when receiver decode fails")
	}
}

func TestRedisLoadManyDoesNotWriteWhenReceiverDecodeFails(t *testing.T) {
	ctx := context.Background()
	cache, client, _ := newRedisTestCache(t)

	loader := func(context.Context, []string) ([]any, error) {
		return []any{"not-an-int"}, nil
	}

	var got int
	receiverMap := map[string]any{"bad-load-many": &got}
	err := cache.LoadMany(ctx, loader, receiverMap, cachespi.NoExpiration)
	if err == nil {
		t.Fatalf("expected decode error")
	}

	exists, err := client.Exists(ctx, "bad-load-many").Result()
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists != 0 {
		t.Fatalf("LoadMany should not write cache when receiver decode fails")
	}
}

func TestSetManyEmptyMapSkipsTTLValidation(t *testing.T) {
	ctx := context.Background()
	ristrettoCache := newRistrettoTestCache(t)
	if err := ristrettoCache.SetMany(ctx, map[string]any{}, -2*time.Second); err != nil {
		t.Fatalf("Ristretto SetMany empty map should be no-op: %v", err)
	}

	redisCache, _, _ := newRedisTestCache(t)
	if err := redisCache.SetMany(ctx, map[string]any{}, -2*time.Second); err != nil {
		t.Fatalf("Redis SetMany empty map should be no-op: %v", err)
	}
}

func TestLoadManyDoesNotDeleteKeysOnLoaderError(t *testing.T) {
	ctx := context.Background()
	loaderErr := errors.New("loader failed")
	loader := func(context.Context, []string) ([]any, error) {
		return nil, loaderErr
	}

	t.Run("ristretto", func(t *testing.T) {
		cache := newRistrettoTestCache(t)
		var got string
		receiverMap := map[string]any{"missing": &got}
		err := cache.LoadMany(ctx, loader, receiverMap, cachespi.NoExpiration)
		if !errors.Is(err, loaderErr) {
			t.Fatalf("expected loader error, got %v", err)
		}
		if _, ok := receiverMap["missing"]; !ok {
			t.Fatalf("loader error should not delete receiverMap key")
		}
	})

	t.Run("redis", func(t *testing.T) {
		cache, _, _ := newRedisTestCache(t)
		var got string
		receiverMap := map[string]any{"missing": &got}
		err := cache.LoadMany(ctx, loader, receiverMap, cachespi.NoExpiration)
		if !errors.Is(err, loaderErr) {
			t.Fatalf("expected loader error, got %v", err)
		}
		if _, ok := receiverMap["missing"]; !ok {
			t.Fatalf("loader error should not delete receiverMap key")
		}
	})
}

func TestRedisCustomCodec(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})
	cache, err := NewRedisCache(RedisConfig{
		Addr:  server.Addr(),
		Codec: prefixCodec{},
	})
	if err != nil {
		t.Fatalf("NewRedisCache: %v", err)
	}

	if err := cache.Set(ctx, "custom-codec", "value", cachespi.NoExpiration); err != nil {
		t.Fatalf("Set: %v", err)
	}
	raw, err := client.Get(ctx, "custom-codec").Result()
	if err != nil {
		t.Fatalf("raw Get: %v", err)
	}
	if raw != "prefix:value" {
		t.Fatalf("unexpected raw value: %q", raw)
	}

	var got string
	if err := cache.Get(ctx, "custom-codec", &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "value" {
		t.Fatalf("unexpected decoded value: %q", got)
	}
}

func TestRedisCloseClosesClient(t *testing.T) {
	ctx := context.Background()
	server := miniredis.RunT(t)

	cache, err := NewRedisCache(RedisConfig{Addr: server.Addr()})
	if err != nil {
		t.Fatalf("NewRedisCache: %v", err)
	}
	if err := cache.Ping(ctx); err != nil {
		t.Fatalf("Ping before Close: %v", err)
	}
	if err := cache.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := cache.Ping(ctx); err == nil {
		t.Fatalf("client should be closed")
	}
}

func newRedisTestCache(t *testing.T) (*RedisCache, *redis.Client, *miniredis.Miniredis) {
	t.Helper()

	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() {
		_ = client.Close()
	})

	cache, err := NewRedisCache(RedisConfig{
		Addr:       server.Addr(),
		DefaultTTL: 5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewRedisCache: %v", err)
	}

	return cache, client, server
}

func newRistrettoTestCache(t *testing.T) *RistrettoCache {
	t.Helper()

	cache, err := NewRistrettoCache(RistrettoConfig{})
	if err != nil {
		t.Fatalf("NewRistrettoCache: %v", err)
	}
	t.Cleanup(func() {
		if err := cache.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	})
	return cache
}

type prefixCodec struct{}

func (prefixCodec) Marshal(value any) ([]byte, error) {
	s, ok := value.(string)
	if !ok {
		return nil, errors.New("prefixCodec only supports string")
	}
	return []byte("prefix:" + s), nil
}

func (prefixCodec) Unmarshal(data []byte, receiver any) error {
	s, ok := receiver.(*string)
	if !ok {
		return errors.New("prefixCodec receiver must be *string")
	}
	const prefix = "prefix:"
	if len(data) < len(prefix) || string(data[:len(prefix)]) != prefix {
		return errors.New("prefixCodec missing prefix")
	}
	*s = string(data[len(prefix):])
	return nil
}
