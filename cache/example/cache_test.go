package example

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachehelper"
	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
	"github.com/alicebob/miniredis/v2"
)

type Product struct {
	ID    int64
	Name  string
	Price float64
}

type exampleCache struct {
	cachespi.Cache
	advanceTime func(time.Duration)
}

type exampleCacheBackend struct {
	name     string
	newCache func(t *testing.T) exampleCache
}

func TestInMemCacheExamples(t *testing.T) {
	runCacheExamples(t, exampleCacheBackend{
		name:     "inmem",
		newCache: newInMemExampleCache,
	})
}

func TestRedisCacheExamples(t *testing.T) {
	runCacheExamples(t, exampleCacheBackend{
		name:     "redis",
		newCache: newRedisExampleCache,
	})
}

func runCacheExamples(t *testing.T, backend exampleCacheBackend) {
	t.Run("SetAndGet", func(t *testing.T) {
		testCacheSetAndGet(t, backend.newCache(t))
	})
	t.Run("Miss", func(t *testing.T) {
		testCacheMiss(t, backend.newCache(t))
	})
	t.Run("NoExpiration", func(t *testing.T) {
		testCacheNoExpiration(t, backend.newCache(t))
	})
	t.Run("CustomTTL", func(t *testing.T) {
		testCacheCustomTTL(t, backend.newCache(t))
	})
	t.Run("SetManyAndGetMany", func(t *testing.T) {
		testCacheSetManyAndGetMany(t, backend.newCache(t))
	})
	t.Run("Delete", func(t *testing.T) {
		testCacheDelete(t, backend.newCache(t))
	})
	t.Run("DeleteMany", func(t *testing.T) {
		testCacheDeleteMany(t, backend.newCache(t))
	})
	t.Run("Load", func(t *testing.T) {
		testCacheLoad(t, backend.newCache(t))
	})
	t.Run("LoadMany", func(t *testing.T) {
		testCacheLoadMany(t, backend.newCache(t))
	})
	t.Run("LoadWithErrorLoader", func(t *testing.T) {
		testCacheLoadWithErrorLoader(t, backend.newCache(t))
	})
	t.Run("Ping", func(t *testing.T) {
		testCachePing(t, backend.newCache(t))
	})
	t.Run("OverwriteExistingKey", func(t *testing.T) {
		testCacheOverwriteExistingKey(t, backend.newCache(t))
	})
	t.Run("DifferentValueTypes", func(t *testing.T) {
		testCacheDifferentValueTypes(t, backend.newCache(t))
	})
}

func newInMemExampleCache(t *testing.T) exampleCache {
	t.Helper()

	cache, err := cachehelper.NewInMemCache(
		cachehelper.WithInMemDefaultTTL(5 * time.Minute),
	)
	assertNoError(t, "NewInMemCache", err)
	t.Cleanup(func() {
		assertNoError(t, "Close in-memory cache", cache.Close(context.Background()))
	})
	return exampleCache{
		Cache:       cache,
		advanceTime: time.Sleep,
	}
}

func newRedisExampleCache(t *testing.T) exampleCache {
	t.Helper()

	server := miniredis.RunT(t)
	cache, err := cachehelper.NewRedisCache(
		cachehelper.WithRedisAddr(server.Addr()),
		cachehelper.WithRedisDefaultTTL(5*time.Minute),
		cachehelper.WithRedisCodec(cachespi.JSONCodec{}),
	)
	assertNoError(t, "NewRedisCache", err)
	t.Cleanup(func() {
		assertNoError(t, "Close redis cache", cache.Close(context.Background()))
	})
	return exampleCache{
		Cache:       cache,
		advanceTime: server.FastForward,
	}
}

// mockLoader simulates a downstream data source.
func mockLoader(data map[string]any) cachespi.DataLoader {
	return func(_ context.Context, keys []string) ([]any, error) {
		results := make([]any, len(keys))
		for i, key := range keys {
			results[i] = data[key] // nil if not found
		}
		return results, nil
	}
}

func testCacheSetAndGet(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	err := cache.Set(ctx, "greeting", "hello world", cachespi.DefaultExpiration)
	assertNoError(t, "Set string", err)

	var greeting string
	err = cache.Get(ctx, "greeting", &greeting)
	assertNoError(t, "Get string", err)
	assertEqual(t, "Get string value", "hello world", greeting)

	err = cache.Set(ctx, "count", 42, cachespi.DefaultExpiration)
	assertNoError(t, "Set int", err)

	var count int
	err = cache.Get(ctx, "count", &count)
	assertNoError(t, "Get int", err)
	assertEqual(t, "Get int value", 42, count)

	product := Product{ID: 1, Name: "Widget", Price: 9.99}
	err = cache.Set(ctx, "product:1", product, cachespi.DefaultExpiration)
	assertNoError(t, "Set struct", err)

	var got Product
	err = cache.Get(ctx, "product:1", &got)
	assertNoError(t, "Get struct", err)
	assertEqual(t, "Get struct value", product, got)

	product2 := &Product{ID: 2, Name: "Gadget", Price: 19.99}
	err = cache.Set(ctx, "product:2", product2, cachespi.DefaultExpiration)
	assertNoError(t, "Set pointer", err)

	var gotPtr *Product
	err = cache.Get(ctx, "product:2", &gotPtr)
	assertNoError(t, "Get pointer", err)
	assertEqual(t, "Get pointer ID", int64(2), gotPtr.ID)
	assertEqual(t, "Get pointer Name", "Gadget", gotPtr.Name)
}

func testCacheMiss(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	var val string
	err := cache.Get(ctx, "non_existent_key", &val)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss, got: %v", err)
	}
}

func testCacheNoExpiration(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	err := cache.Set(ctx, "permanent", "I never expire", cachespi.NoExpiration)
	assertNoError(t, "Set with NoExpiration", err)

	var val string
	err = cache.Get(ctx, "permanent", &val)
	assertNoError(t, "Get permanent", err)
	assertEqual(t, "Get permanent value", "I never expire", val)
}

func testCacheCustomTTL(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	err := cache.Set(ctx, "short_lived", "bye soon", 200*time.Millisecond)
	assertNoError(t, "Set with short TTL", err)

	var val string
	err = cache.Get(ctx, "short_lived", &val)
	assertNoError(t, "Get before expiry", err)
	assertEqual(t, "Get before expiry value", "bye soon", val)

	cache.advanceTime(500 * time.Millisecond)

	err = cache.Get(ctx, "short_lived", &val)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss after expiry, got: %v (val=%v)", err, val)
	}
}

func testCacheSetManyAndGetMany(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	values := map[string]any{
		"key_a": "alpha",
		"key_b": "beta",
		"key_c": "gamma",
	}
	err := cache.SetMany(ctx, values, cachespi.DefaultExpiration)
	assertNoError(t, "SetMany", err)

	var a, b, d string
	receiverMap := map[string]any{
		"key_a":       &a,
		"key_b":       &b,
		"key_missing": &d,
	}
	err = cache.GetMany(ctx, receiverMap)
	assertNoError(t, "GetMany", err)

	if _, ok := receiverMap["key_missing"]; ok {
		t.Fatalf("expected key_missing to be removed from receiverMap")
	}
	assertEqual(t, "GetMany key_a", "alpha", a)
	assertEqual(t, "GetMany key_b", "beta", b)
}

func testCacheDelete(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	assertNoError(t, "Set to_delete", cache.Set(ctx, "to_delete", "doomed", cachespi.DefaultExpiration))

	err := cache.Delete(ctx, "to_delete")
	assertNoError(t, "Delete existing", err)

	var val string
	err = cache.Get(ctx, "to_delete", &val)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss after Delete, got: %v", err)
	}

	err = cache.Delete(ctx, "never_existed")
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss for non-existent key, got: %v", err)
	}
}

func testCacheDeleteMany(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	assertNoError(t, "Set dm_1", cache.Set(ctx, "dm_1", "one", cachespi.DefaultExpiration))
	assertNoError(t, "Set dm_2", cache.Set(ctx, "dm_2", "two", cachespi.DefaultExpiration))
	assertNoError(t, "Set dm_3", cache.Set(ctx, "dm_3", "three", cachespi.DefaultExpiration))

	err := cache.DeleteMany(ctx, []string{"dm_1", "dm_3"})
	assertNoError(t, "DeleteMany", err)

	var val string
	err = cache.Get(ctx, "dm_1", &val)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("dm_1 should be deleted")
	}

	err = cache.Get(ctx, "dm_2", &val)
	assertNoError(t, "dm_2 should still exist", err)
	assertEqual(t, "dm_2 value", "two", val)

	err = cache.Get(ctx, "dm_3", &val)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("dm_3 should be deleted")
	}
}

func testCacheLoad(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	loader := mockLoader(map[string]any{
		"user:100": &Product{ID: 100, Name: "Loaded Product", Price: 49.99},
	})

	var product *Product
	err := cache.Load(ctx, loader, "user:100", &product, cachespi.DefaultExpiration)
	assertNoError(t, "Load from loader", err)
	assertEqual(t, "Loaded product name", "Loaded Product", product.Name)

	var product2 *Product
	err = cache.Load(ctx, loader, "user:100", &product2, cachespi.DefaultExpiration)
	assertNoError(t, "Load from cache", err)
	assertEqual(t, "Cached product name", "Loaded Product", product2.Name)

	var missing *Product
	err = cache.Load(ctx, loader, "user:999", &missing, cachespi.DefaultExpiration)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss for missing loader key, got: %v", err)
	}
}

func testCacheLoadMany(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	assertNoError(t, "Set item:1", cache.Set(ctx, "item:1", "cached_one", cachespi.DefaultExpiration))

	loader := mockLoader(map[string]any{
		"item:2": "loaded_two",
		"item:3": "loaded_three",
	})

	var v1, v2, v3, v4 string
	receiverMap := map[string]any{
		"item:1": &v1,
		"item:2": &v2,
		"item:3": &v3,
		"item:4": &v4,
	}

	err := cache.LoadMany(ctx, loader, receiverMap, cachespi.DefaultExpiration)
	assertNoError(t, "LoadMany", err)

	assertEqual(t, "item:1", "cached_one", v1)
	assertEqual(t, "item:2", "loaded_two", v2)
	assertEqual(t, "item:3", "loaded_three", v3)

	if _, ok := receiverMap["item:4"]; ok {
		t.Fatalf("item:4 should be removed from receiverMap")
	}

	var v2Cached string
	err = cache.Get(ctx, "item:2", &v2Cached)
	assertNoError(t, "Get item:2 after LoadMany", err)
	assertEqual(t, "item:2 now cached", "loaded_two", v2Cached)
}

func testCacheLoadWithErrorLoader(t *testing.T, cache exampleCache) {
	ctx := context.Background()
	downstreamErr := errors.New("downstream unavailable")
	errLoader := func(_ context.Context, _ []string) ([]any, error) {
		return nil, downstreamErr
	}

	var val string
	err := cache.Load(ctx, errLoader, "fail_key", &val, cachespi.DefaultExpiration)
	if !errors.Is(err, downstreamErr) {
		t.Fatalf("expected downstream error, got: %v", err)
	}
}

func testCachePing(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	err := cache.Ping(ctx)
	assertNoError(t, "Ping", err)
}

func testCacheOverwriteExistingKey(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	assertNoError(t, "Set overwrite original", cache.Set(ctx, "overwrite", "original", cachespi.DefaultExpiration))
	assertNoError(t, "Set overwrite updated", cache.Set(ctx, "overwrite", "updated", cachespi.DefaultExpiration))

	var val string
	err := cache.Get(ctx, "overwrite", &val)
	assertNoError(t, "Get overwritten key", err)
	assertEqual(t, "Overwritten value", "updated", val)
}

func testCacheDifferentValueTypes(t *testing.T, cache exampleCache) {
	ctx := context.Background()

	assertNoError(t, "Set float64", cache.Set(ctx, "pi", 3.14159, cachespi.DefaultExpiration))
	var f float64
	err := cache.Get(ctx, "pi", &f)
	assertNoError(t, "Get float64", err)
	assertEqual(t, "float64 value", 3.14159, f)

	assertNoError(t, "Set bool", cache.Set(ctx, "flag", true, cachespi.DefaultExpiration))
	var b bool
	err = cache.Get(ctx, "flag", &b)
	assertNoError(t, "Get bool", err)
	assertEqual(t, "bool value", true, b)

	nums := []int{1, 2, 3, 4, 5}
	assertNoError(t, "Set slice", cache.Set(ctx, "nums", nums, cachespi.DefaultExpiration))
	var gotNums []int
	err = cache.Get(ctx, "nums", &gotNums)
	assertNoError(t, "Get slice", err)
	assertEqual(t, "slice length", len(nums), len(gotNums))
	for i := range nums {
		assertEqual(t, fmt.Sprintf("slice[%d]", i), nums[i], gotNums[i])
	}

	m := map[string]int{"a": 1, "b": 2}
	assertNoError(t, "Set map", cache.Set(ctx, "mymap", m, cachespi.DefaultExpiration))
	var gotMap map[string]int
	err = cache.Get(ctx, "mymap", &gotMap)
	assertNoError(t, "Get map", err)
	assertEqual(t, "map[a]", 1, gotMap["a"])
	assertEqual(t, "map[b]", 2, gotMap["b"])
}

func assertNoError(t *testing.T, label string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("[%s] unexpected error: %v", label, err)
	}
}

func assertEqual[T comparable](t *testing.T, label string, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("[%s] expected %v, got %v", label, expected, actual)
	}
}
