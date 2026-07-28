package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
)

const (
	staffIDKey ctxKey = iota + 10
	hospitalKey
)

// Auth validates a Bearer JWT and injects staff_id + hospital into the context.
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			abortUnauthorized(c)
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			abortUnauthorized(c)
			return
		}

		claims, err := platform.Parse(parts[1], jwtSecret)
		if err != nil {
			abortUnauthorized(c)
			return
		}

		ctx := c.Request.Context()
		ctx = context.WithValue(ctx, staffIDKey, claims.StaffID)
		ctx = context.WithValue(ctx, hospitalKey, claims.Hospital)
		c.Request = c.Request.WithContext(ctx)
		c.Set("staff_id", claims.StaffID)
		c.Set("hospital", claims.Hospital)
		c.Next()
	}
}

// StaffIDFromContext returns the authenticated staff id, or empty.
func StaffIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(staffIDKey).(string); ok {
		return v
	}
	return ""
}

// HospitalFromContext returns the authenticated hospital claim, or empty.
func HospitalFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(hospitalKey).(string); ok {
		return v
	}
	return ""
}

func abortUnauthorized(c *gin.Context) {
	rid := RequestIDFromContext(c.Request.Context())
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"error": gin.H{
			"code":       "UNAUTHORIZED",
			"message":    "authentication required",
			"request_id": rid,
		},
	})
}
