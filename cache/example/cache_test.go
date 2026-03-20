package example

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/MrMiaoMIMI/goshared/cache/cachehelper"
	"github.com/MrMiaoMIMI/goshared/cache/cachespi"
)

// ==================== Test Models ====================

type Product struct {
	ID    int64
	Name  string
	Price float64
}

// ==================== Helpers ====================

func testNewCache() cachespi.Cache {
	return cachehelper.NewInMemCache(
		cachehelper.WithDefaultTTL(5 * time.Minute),
	)
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

// ==================== Tests ====================

func Test_Cache_SetAndGet(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	// Example 1: Set and Get a string
	err := cache.Set(ctx, "greeting", "hello world", cachespi.DefaultExpiration)
	assertNoError(t, "Set string", err)

	var greeting string
	err = cache.Get(ctx, "greeting", &greeting)
	assertNoError(t, "Get string", err)
	assertEqual(t, "Get string value", "hello world", greeting)

	// Example 2: Set and Get an integer
	err = cache.Set(ctx, "count", 42, cachespi.DefaultExpiration)
	assertNoError(t, "Set int", err)

	var count int
	err = cache.Get(ctx, "count", &count)
	assertNoError(t, "Get int", err)
	assertEqual(t, "Get int value", 42, count)

	// Example 3: Set and Get a struct (by value)
	product := Product{ID: 1, Name: "Widget", Price: 9.99}
	err = cache.Set(ctx, "product:1", product, cachespi.DefaultExpiration)
	assertNoError(t, "Set struct", err)

	var got Product
	err = cache.Get(ctx, "product:1", &got)
	assertNoError(t, "Get struct", err)
	assertEqual(t, "Get struct value", product, got)

	// Example 4: Set and Get a pointer to struct
	product2 := &Product{ID: 2, Name: "Gadget", Price: 19.99}
	err = cache.Set(ctx, "product:2", product2, cachespi.DefaultExpiration)
	assertNoError(t, "Set pointer", err)

	var gotPtr *Product
	err = cache.Get(ctx, "product:2", &gotPtr)
	assertNoError(t, "Get pointer", err)
	assertEqual(t, "Get pointer ID", int64(2), gotPtr.ID)
	assertEqual(t, "Get pointer Name", "Gadget", gotPtr.Name)
}

func Test_Cache_Miss(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	var val string
	err := cache.Get(ctx, "non_existent_key", &val)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss, got: %v", err)
	}
	t.Logf("Cache miss returned expected error: %v", err)
}

func Test_Cache_NoExpiration(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	err := cache.Set(ctx, "permanent", "I never expire", cachespi.NoExpiration)
	assertNoError(t, "Set with NoExpiration", err)

	var val string
	err = cache.Get(ctx, "permanent", &val)
	assertNoError(t, "Get permanent", err)
	assertEqual(t, "Get permanent value", "I never expire", val)
}

func Test_Cache_CustomTTL(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	err := cache.Set(ctx, "short_lived", "bye soon", 500*time.Millisecond)
	assertNoError(t, "Set with short TTL", err)

	var val string
	err = cache.Get(ctx, "short_lived", &val)
	assertNoError(t, "Get before expiry", err)
	assertEqual(t, "Get before expiry value", "bye soon", val)

	time.Sleep(1 * time.Second)

	err = cache.Get(ctx, "short_lived", &val)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss after expiry, got: %v (val=%v)", err, val)
	}
	t.Logf("Key expired as expected")
}

func Test_Cache_SetManyAndGetMany(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	// SetMany
	values := map[string]any{
		"key_a": "alpha",
		"key_b": "beta",
		"key_c": "gamma",
	}
	err := cache.SetMany(ctx, values, cachespi.DefaultExpiration)
	assertNoError(t, "SetMany", err)

	// GetMany - include one key that exists and one that doesn't
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
	t.Logf("GetMany returned %d keys (missing keys removed)", len(receiverMap))
}

func Test_Cache_Delete(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	_ = cache.Set(ctx, "to_delete", "doomed", cachespi.DefaultExpiration)

	err := cache.Delete(ctx, "to_delete")
	assertNoError(t, "Delete existing", err)

	var val string
	err = cache.Get(ctx, "to_delete", &val)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss after Delete, got: %v", err)
	}

	// Delete non-existent key
	err = cache.Delete(ctx, "never_existed")
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss for non-existent key, got: %v", err)
	}
}

func Test_Cache_DeleteMany(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	_ = cache.Set(ctx, "dm_1", "one", cachespi.DefaultExpiration)
	_ = cache.Set(ctx, "dm_2", "two", cachespi.DefaultExpiration)
	_ = cache.Set(ctx, "dm_3", "three", cachespi.DefaultExpiration)

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

func Test_Cache_Load(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	loader := mockLoader(map[string]any{
		"user:100": &Product{ID: 100, Name: "Loaded Product", Price: 49.99},
	})

	// Load from loader (cache miss → loader → cache set + return)
	var product *Product
	err := cache.Load(ctx, loader, "user:100", &product, cachespi.DefaultExpiration)
	assertNoError(t, "Load from loader", err)
	assertEqual(t, "Loaded product name", "Loaded Product", product.Name)

	// Load again (should hit cache, not loader)
	var product2 *Product
	err = cache.Load(ctx, loader, "user:100", &product2, cachespi.DefaultExpiration)
	assertNoError(t, "Load from cache", err)
	assertEqual(t, "Cached product name", "Loaded Product", product2.Name)

	// Load a key not in loader
	var missing *Product
	err = cache.Load(ctx, loader, "user:999", &missing, cachespi.DefaultExpiration)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("expected ErrCacheMiss for missing loader key, got: %v", err)
	}
}

func Test_Cache_LoadMany(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	// Pre-populate one key
	_ = cache.Set(ctx, "item:1", "cached_one", cachespi.DefaultExpiration)

	loader := mockLoader(map[string]any{
		"item:2": "loaded_two",
		"item:3": "loaded_three",
	})

	var v1, v2, v3, v4 string
	receiverMap := map[string]any{
		"item:1": &v1, // from cache
		"item:2": &v2, // from loader
		"item:3": &v3, // from loader
		"item:4": &v4, // not in cache or loader
	}

	err := cache.LoadMany(ctx, loader, receiverMap, cachespi.DefaultExpiration)
	assertNoError(t, "LoadMany", err)

	assertEqual(t, "item:1 (from cache)", "cached_one", v1)
	assertEqual(t, "item:2 (from loader)", "loaded_two", v2)
	assertEqual(t, "item:3 (from loader)", "loaded_three", v3)

	if _, ok := receiverMap["item:4"]; ok {
		t.Fatalf("item:4 should be removed from receiverMap (not in cache or loader)")
	}
	t.Logf("LoadMany returned %d keys", len(receiverMap))

	// Verify loaded items are now cached
	var v2Cached string
	err = cache.Get(ctx, "item:2", &v2Cached)
	assertNoError(t, "Get item:2 after LoadMany", err)
	assertEqual(t, "item:2 now cached", "loaded_two", v2Cached)
}

func Test_Cache_LoadWithErrorLoader(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	errLoader := func(_ context.Context, _ []string) ([]any, error) {
		return nil, fmt.Errorf("downstream unavailable")
	}

	var val string
	err := cache.Load(ctx, errLoader, "fail_key", &val, cachespi.DefaultExpiration)
	if err == nil || err.Error() != "downstream unavailable" {
		t.Fatalf("expected downstream error, got: %v", err)
	}
	t.Logf("Load with error loader returned expected error: %v", err)
}

func Test_Cache_Flush(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	_ = cache.Set(ctx, "flush_1", "one", cachespi.DefaultExpiration)
	_ = cache.Set(ctx, "flush_2", "two", cachespi.DefaultExpiration)

	err := cache.Flush(ctx)
	assertNoError(t, "Flush", err)

	var val string
	err = cache.Get(ctx, "flush_1", &val)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("flush_1 should be gone after Flush")
	}
	err = cache.Get(ctx, "flush_2", &val)
	if !errors.Is(err, cachespi.ErrCacheMiss) {
		t.Fatalf("flush_2 should be gone after Flush")
	}
	t.Logf("All keys flushed successfully")
}

func Test_Cache_Ping(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	err := cache.Ping(ctx)
	assertNoError(t, "Ping", err)
	t.Logf("Ping succeeded")
}

func Test_Cache_OverwriteExistingKey(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	_ = cache.Set(ctx, "overwrite", "original", cachespi.DefaultExpiration)
	_ = cache.Set(ctx, "overwrite", "updated", cachespi.DefaultExpiration)

	var val string
	err := cache.Get(ctx, "overwrite", &val)
	assertNoError(t, "Get overwritten key", err)
	assertEqual(t, "Overwritten value", "updated", val)
}

func Test_Cache_DifferentValueTypes(t *testing.T) {
	ctx := context.Background()
	cache := testNewCache()

	// float64
	_ = cache.Set(ctx, "pi", 3.14159, cachespi.DefaultExpiration)
	var f float64
	err := cache.Get(ctx, "pi", &f)
	assertNoError(t, "Get float64", err)
	assertEqual(t, "float64 value", 3.14159, f)

	// bool
	_ = cache.Set(ctx, "flag", true, cachespi.DefaultExpiration)
	var b bool
	err = cache.Get(ctx, "flag", &b)
	assertNoError(t, "Get bool", err)
	assertEqual(t, "bool value", true, b)

	// slice
	nums := []int{1, 2, 3, 4, 5}
	_ = cache.Set(ctx, "nums", nums, cachespi.DefaultExpiration)
	var gotNums []int
	err = cache.Get(ctx, "nums", &gotNums)
	assertNoError(t, "Get slice", err)
	assertEqual(t, "slice length", len(nums), len(gotNums))
	for i := range nums {
		assertEqual(t, fmt.Sprintf("slice[%d]", i), nums[i], gotNums[i])
	}

	// map
	m := map[string]int{"a": 1, "b": 2}
	_ = cache.Set(ctx, "mymap", m, cachespi.DefaultExpiration)
	var gotMap map[string]int
	err = cache.Get(ctx, "mymap", &gotMap)
	assertNoError(t, "Get map", err)
	assertEqual(t, "map[a]", 1, gotMap["a"])
	assertEqual(t, "map[b]", 2, gotMap["b"])
}

// ==================== Assertion Helpers ====================

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
