// Package httpkit holds the response envelope and error-mapping helpers
// shared by every context's HTTP transport package. It depends only on
// platform and apperr — never on a specific context's domain or handlers —
// so both staff/transport/http and patient/transport/http can import it
// without creating a cycle back through the router.
package httpkit

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// WriteData writes a success envelope with a single resource.
func WriteData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

// WriteCollection writes a success envelope with a list and a count.
func WriteCollection(c *gin.Context, status int, data any, count int) {
	c.JSON(status, gin.H{
		"data": data,
		"meta": gin.H{"count": count},
	})
}

// WriteError writes the standard error envelope.
func WriteError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": errorBody{
			Code:      code,
			Message:   message,
			RequestID: middleware.RequestIDFromContext(c.Request.Context()),
		},
	})
}

// WriteBindError writes the 400/413 response for a failed ShouldBindJSON call.
func WriteBindError(c *gin.Context, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		WriteError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
		return
	}
	WriteError(c, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
}

// MapError translates a service-layer sentinel error into an HTTP response.
func MapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, apperr.ErrInvalidInput):
		WriteError(c, http.StatusBadRequest, "INVALID_INPUT", safeMessage(err, "invalid input"))
	case errors.Is(err, platform.ErrUnauthorized):
		WriteError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
	case errors.Is(err, platform.ErrNotFound):
		WriteError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, platform.ErrConflict):
		WriteError(c, http.StatusConflict, "CONFLICT", "resource already exists")
	case errors.Is(err, platform.ErrRateLimited):
		WriteError(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
	case errors.Is(err, platform.ErrPayloadTooLarge):
		WriteError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
	case errors.Is(err, platform.ErrUpstream):
		WriteError(c, http.StatusBadGateway, "BAD_GATEWAY", "upstream service unavailable")
	default:
		WriteError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}

// safeMessage strips the sentinel prefix from an apperr.ErrInvalidInput chain
// so the client sees "national_id or passport_id required" instead of
// "invalid input: national_id or passport_id required". Any other error
// falls back to a generic, PII-free message.
func safeMessage(err error, fallback string) string {
	const prefix = "invalid input: "
	msg := err.Error()
	if len(msg) > len(prefix) && msg[:len(prefix)] == prefix {
		return msg[len(prefix):]
	}
	return fallback
}
