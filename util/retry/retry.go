// Package retry provides a generic retry mechanism with configurable backoff strategies.
package retry

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// Config configures the retry behavior.
type Config struct {
	MaxAttempts int
	InitDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64           // backoff multiplier (1.0 = constant, 2.0 = exponential)
	Jitter      bool              // add random jitter to delay
	RetryIf     func(error) bool  // only retry if this returns true; nil means always retry
}

// DefaultConfig returns a config with 3 attempts, exponential backoff starting at 100ms.
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		InitDelay:   100 * time.Millisecond,
		MaxDelay:    10 * time.Second,
		Multiplier:  2.0,
		Jitter:      true,
	}
}

// Do retries fn until it succeeds, max attempts are exhausted, or ctx is cancelled.
// Returns the last error if all attempts fail.
func Do(ctx context.Context, cfg Config, fn func(ctx context.Context) error) error {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.InitDelay <= 0 {
		cfg.InitDelay = 100 * time.Millisecond
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 1.0
	}

	var lastErr error
	delay := cfg.InitDelay

	for attempt := range cfg.MaxAttempts {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return lastErr
			}
			return err
		}

		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}

		if cfg.RetryIf != nil && !cfg.RetryIf(lastErr) {
			return lastErr
		}

		if attempt < cfg.MaxAttempts-1 {
			sleepDuration := delay
			if cfg.Jitter {
				jitter := time.Duration(rand.Float64() * float64(sleepDuration) * 0.5)
				sleepDuration += jitter
			}

			select {
			case <-ctx.Done():
				return lastErr
			case <-time.After(sleepDuration):
			}

			delay = time.Duration(float64(delay) * cfg.Multiplier)
			if cfg.MaxDelay > 0 && delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
	}

	return lastErr
}

// DoWithResult retries fn until it succeeds and returns both the result and error.
func DoWithResult[T any](ctx context.Context, cfg Config, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	err := Do(ctx, cfg, func(ctx context.Context) error {
		var fnErr error
		result, fnErr = fn(ctx)
		return fnErr
	})
	return result, err
}

// Simple retries fn up to maxAttempts with constant delay.
func Simple(ctx context.Context, maxAttempts int, delay time.Duration, fn func(ctx context.Context) error) error {
	return Do(ctx, Config{
		MaxAttempts: maxAttempts,
		InitDelay:   delay,
		Multiplier:  1.0,
		Jitter:      false,
	}, fn)
}

// Exponential retries fn with exponential backoff.
func Exponential(ctx context.Context, maxAttempts int, initDelay time.Duration, fn func(ctx context.Context) error) error {
	return Do(ctx, Config{
		MaxAttempts: maxAttempts,
		InitDelay:   initDelay,
		MaxDelay:    time.Duration(float64(initDelay) * math.Pow(2.0, float64(maxAttempts))),
		Multiplier:  2.0,
		Jitter:      true,
	}, fn)
}
