package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// BodyLimit caps the request body at 1 MiB and sets nosniff.
func BodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
		}
		c.Next()
		if c.Writer.Status() == http.StatusRequestEntityTooLarge {
			return
		}
	}
}

// HandlePayloadTooLarge recovers from MaxBytesReader rejection.
// Call after BodyLimit; Gin surfaces MaxBytesError via bind failures — this
// middleware also checks Content-Length early.
func HandlePayloadTooLarge() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBodyBytes {
			rid := RequestIDFromContext(c.Request.Context())
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": gin.H{
					"code":       "PAYLOAD_TOO_LARGE",
					"message":    "request body too large",
					"request_id": rid,
				},
			})
			return
		}
		c.Next()
	}
}
