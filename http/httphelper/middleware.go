package httphelper

import (
	"net/http"
	"time"

	"github.com/MrMiaoMIMI/goshared/logger"
	"github.com/MrMiaoMIMI/goshared/util/random"
	"github.com/gin-gonic/gin"
)

const (
	HeaderRequestID = "X-Request-ID"
)

// RequestID is a Gin middleware that generates a unique request ID (X-Request-ID)
// and sets it into the context's trace_id for logging.
// If the incoming request already has an X-Request-ID header, it is reused.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(HeaderRequestID)
		if requestID == "" {
			requestID = random.GenRandomHashKey()[:16]
		}
		c.Set(string(logger.TraceIDKey), requestID)
		c.Writer.Header().Set(HeaderRequestID, requestID)

		ctx := logger.SetTraceID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// AccessLog is a Gin middleware that logs each request with method, path, status, and latency.
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []logger.Field{
			logger.String("method", c.Request.Method),
			logger.String("path", path),
			logger.Int("status", status),
			logger.Duration("latency", latency),
			logger.String("client_ip", c.ClientIP()),
		}
		if query != "" {
			fields = append(fields, logger.String("query", query))
		}
		if len(c.Errors) > 0 {
			fields = append(fields, logger.String("errors", c.Errors.String()))
		}

		if status >= 500 {
			logger.Error(c.Request.Context(), "request", fields...)
		} else if status >= 400 {
			logger.Warn(c.Request.Context(), "request", fields...)
		} else {
			logger.Info(c.Request.Context(), "request", fields...)
		}
	}
}

// Recovery is a Gin middleware that recovers from panics and returns a 500 JSON response.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error(c.Request.Context(), "panic recovered",
					logger.Any("error", err),
					logger.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    50000001,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}

// CORS is a Gin middleware that handles Cross-Origin Resource Sharing.
// Pass "*" to allow all origins. If no origins are specified, no CORS
// headers are set (deny by default).
func CORS(allowOrigins ...string) gin.HandlerFunc {
	originSet := make(map[string]struct{}, len(allowOrigins))
	allowAll := false
	for _, o := range allowOrigins {
		if o == "*" {
			allowAll = true
		}
		originSet[o] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		_, allowed := originSet[origin]
		if allowAll || allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
