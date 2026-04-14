// Package syncutil provides concurrency utilities for safe goroutine management.
package syncutil

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// SafeGo runs fn in a new goroutine with panic recovery.
// If the goroutine panics, onPanic is called with the recovered value.
// If onPanic is nil, the panic is silently discarded.
func SafeGo(fn func(), onPanic func(recovered any)) {
	go func() {
		defer func() {
			if r := recover(); r != nil && onPanic != nil {
				onPanic(r)
			}
		}()
		fn()
	}()
}

// SafeGoWithStack runs fn in a new goroutine with panic recovery.
// On panic, calls onPanic with the recovered value and a stack trace string.
func SafeGoWithStack(fn func(), onPanic func(recovered any, stack string)) {
	go func() {
		defer func() {
			if r := recover(); r != nil && onPanic != nil {
				onPanic(r, string(debug.Stack()))
			}
		}()
		fn()
	}()
}

// Parallel runs all functions concurrently and waits for them all to complete.
// Returns the first non-nil error (if any).
func Parallel(fns ...func() error) error {
	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		retErr  error
	)
	wg.Add(len(fns))
	for _, fn := range fns {
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				errOnce.Do(func() { retErr = err })
			}
		}()
	}
	wg.Wait()
	return retErr
}

// ParallelAll runs all functions concurrently and returns all errors.
func ParallelAll(fns ...func() error) []error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	wg.Add(len(fns))
	for _, fn := range fns {
		go func() {
			defer wg.Done()
			if err := fn(); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return errs
}

// Semaphore is a counting semaphore for limiting concurrency.
type Semaphore struct {
	ch chan struct{}
}

// NewSemaphore creates a semaphore with the given maximum concurrency.
// Panics if maxConcurrency is less than 1.
func NewSemaphore(maxConcurrency int) *Semaphore {
	if maxConcurrency < 1 {
		panic("syncutil: semaphore maxConcurrency must be >= 1")
	}
	return &Semaphore{ch: make(chan struct{}, maxConcurrency)}
}

// Acquire blocks until a slot is available or ctx is cancelled.
func (s *Semaphore) Acquire(ctx context.Context) error {
	select {
	case s.ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire attempts to acquire a slot without blocking.
// Returns true if acquired, false otherwise.
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.ch <- struct{}{}:
		return true
	default:
		return false
	}
}

// Release releases a slot.
func (s *Semaphore) Release() {
	<-s.ch
}

// OnceValue lazily computes a value exactly once using fn.
// Subsequent calls return the cached result. Thread-safe.
func OnceValue[T any](fn func() T) func() T {
	var (
		once sync.Once
		val  T
	)
	return func() T {
		once.Do(func() { val = fn() })
		return val
	}
}

// OnceValueErr lazily computes a value exactly once using fn.
// If fn returns an error, subsequent calls will retry.
func OnceValueErr[T any](fn func() (T, error)) func() (T, error) {
	var (
		mu   sync.Mutex
		done bool
		val  T
	)
	return func() (T, error) {
		mu.Lock()
		defer mu.Unlock()
		if done {
			return val, nil
		}
		result, err := fn()
		if err != nil {
			return result, err
		}
		val = result
		done = true
		return val, nil
	}
}

// Fan runs fn for each item in items concurrently with maxConcurrency limit.
// Returns the first error encountered (or nil).
func Fan[T any](ctx context.Context, maxConcurrency int, items []T, fn func(context.Context, T) error) error {
	sem := NewSemaphore(maxConcurrency)
	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		retErr  error
	)

	for _, item := range items {
		if err := sem.Acquire(ctx); err != nil {
			return err
		}
		wg.Add(1)
		go func() {
			defer func() {
				sem.Release()
				wg.Done()
			}()
			if err := fn(ctx, item); err != nil {
				errOnce.Do(func() { retErr = err })
			}
		}()
	}
	wg.Wait()
	return retErr
}

// Debounce returns a function that delays invoking fn until after wait duration
// has elapsed since the last time the returned function was called.
func Debounce(fn func(), wait time.Duration) func() {
	var mu sync.Mutex
	var timer *time.Timer
	return func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(wait, fn)
	}
}

// Pool is a typed wrapper around sync.Pool.
type Pool[T any] struct {
	pool sync.Pool
}

// NewPool creates a typed sync.Pool with the given factory function.
func NewPool[T any](factory func() T) *Pool[T] {
	return &Pool[T]{
		pool: sync.Pool{
			New: func() any { return factory() },
		},
	}
}

// Get retrieves an item from the pool.
func (p *Pool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put returns an item to the pool.
func (p *Pool[T]) Put(v T) {
	p.pool.Put(v)
}

// WaitGroupTimeout waits for a WaitGroup with a timeout.
// Returns an error if the timeout expires before all goroutines complete.
func WaitGroupTimeout(wg *sync.WaitGroup, timeout time.Duration) error {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("syncutil: WaitGroup timed out after %v", timeout)
	}
}
