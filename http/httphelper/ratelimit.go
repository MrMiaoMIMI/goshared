package httphelper

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimitConfig configures the rate limiter middleware.
type RateLimitConfig struct {
	Rate     int           // max requests per window
	Window   time.Duration // time window
	KeyFunc  func(*gin.Context) string
	Response gin.H
}

type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	rate     int
	window   time.Duration
}

type visitor struct {
	count    int
	resetAt  time.Time
}

func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	v, exists := rl.visitors[key]
	if !exists || now.After(v.resetAt) {
		rl.visitors[key] = &visitor{count: 1, resetAt: now.Add(rl.window)}
		return true
	}
	if v.count >= rl.rate {
		return false
	}
	v.count++
	return true
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(rl.window)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, v := range rl.visitors {
			if now.After(v.resetAt) {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit creates a Gin rate limiter middleware.
// Default: rate=100 per 1 minute, keyed by client IP.
//
// Example:
//
//	r.Use(httphelper.RateLimit(httphelper.RateLimitConfig{
//	    Rate:   60,
//	    Window: time.Minute,
//	}))
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	if cfg.Rate <= 0 {
		cfg.Rate = 100
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *gin.Context) string { return c.ClientIP() }
	}
	if cfg.Response == nil {
		cfg.Response = gin.H{
			"code":    42900000,
			"message": "too many requests",
		}
	}

	limiter := newRateLimiter(cfg.Rate, cfg.Window)

	return func(c *gin.Context) {
		key := cfg.KeyFunc(c)
		if !limiter.allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, cfg.Response)
			return
		}
		c.Next()
	}
}
