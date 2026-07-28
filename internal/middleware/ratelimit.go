package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// FixedWindowLimiter is a simple in-memory fixed-window rate limiter.
type FixedWindowLimiter struct {
	mu       sync.Mutex
	window   time.Duration
	limit    int
	counters map[string]*windowCounter
	now      func() time.Time
}

type windowCounter struct {
	windowStart time.Time
	count       int
}

// NewFixedWindowLimiter creates a limiter allowing `limit` events per `window`.
func NewFixedWindowLimiter(limit int, window time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{
		window:   window,
		limit:    limit,
		counters: make(map[string]*windowCounter),
		now:      time.Now,
	}
}

// Allow reports whether key is within its quota and increments the counter.
func (l *FixedWindowLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	c, ok := l.counters[key]
	if !ok || now.Sub(c.windowStart) >= l.window {
		l.counters[key] = &windowCounter{windowStart: now, count: 1}
		return true
	}
	if c.count >= l.limit {
		return false
	}
	c.count++
	return true
}

// RateLimitByIP rejects requests when the client IP exceeds the limit.
func RateLimitByIP(limiter *FixedWindowLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !limiter.Allow(ip) {
			abortRateLimited(c)
			return
		}
		c.Next()
	}
}

// RateLimitByStaff rejects requests when the authenticated staff exceeds the limit.
func RateLimitByStaff(limiter *FixedWindowLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		staffID := StaffIDFromContext(c.Request.Context())
		if staffID == "" {
			staffID = c.ClientIP()
		}
		if !limiter.Allow(staffID) {
			abortRateLimited(c)
			return
		}
		c.Next()
	}
}

func abortRateLimited(c *gin.Context) {
	rid := RequestIDFromContext(c.Request.Context())
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error": gin.H{
			"code":       "RATE_LIMITED",
			"message":    "too many requests",
			"request_id": rid,
		},
	})
}
