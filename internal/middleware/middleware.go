package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ctxKey int

const (
	requestIDKey ctxKey = iota
	loggerKey
)

const requestIDHeader = "X-Request-ID"

// RequestID propagates or generates X-Request-ID and stores it on the context.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if id == "" {
			id = uuid.NewString()
		}
		c.Writer.Header().Set(requestIDHeader, id)
		ctx := context.WithValue(c.Request.Context(), requestIDKey, id)
		c.Request = c.Request.WithContext(ctx)
		c.Set("request_id", id)
		c.Next()
	}
}

// RequestIDFromContext returns the request id if present.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// Recovery logs panics with stack and returns a generic INTERNAL_ERROR envelope.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				rid := RequestIDFromContext(c.Request.Context())
				log.Error("panic recovered",
					"request_id", rid,
					"error", rec,
					"stack", string(debug.Stack()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":       "INTERNAL_ERROR",
						"message":    "an unexpected error occurred",
						"request_id": rid,
					},
				})
			}
		}()
		c.Next()
	}
}

// Logger emits one access log line per request (skips health probes).
func Logger(base *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/healthz" || c.Request.URL.Path == "/readyz" {
			c.Next()
			return
		}

		start := time.Now()
		rid := RequestIDFromContext(c.Request.Context())
		reqLog := base.With("request_id", rid)
		ctx := context.WithValue(c.Request.Context(), loggerKey, reqLog)
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		status := c.Writer.Status()
		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if status >= 500 {
			reqLog.Error("request completed", attrs...)
		} else if status >= 400 {
			reqLog.Warn("request completed", attrs...)
		} else {
			reqLog.Info("request completed", attrs...)
		}
	}
}

// Timeout cancels the request context after d.
func Timeout(d time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), d)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
