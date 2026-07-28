package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
)

type errorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// writeData writes a success envelope with a single resource.
func writeData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data})
}

// writeCollection writes a success envelope with a list and count meta.
func writeCollection(c *gin.Context, status int, data any, count int) {
	c.JSON(status, gin.H{
		"data": data,
		"meta": gin.H{"count": count},
	})
}

// writeError writes a standard error envelope.
func writeError(c *gin.Context, status int, code, message string) {
	rid := middleware.RequestIDFromContext(c.Request.Context())
	c.JSON(status, gin.H{
		"error": errorBody{
			Code:      code,
			Message:   message,
			RequestID: rid,
		},
	})
}

// mapError translates known sentinel errors to HTTP responses.
func mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, platform.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "INVALID_INPUT", safeMessage(err, "invalid input"))
	case errors.Is(err, platform.ErrUnauthorized):
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
	case errors.Is(err, platform.ErrNotFound):
		writeError(c, http.StatusNotFound, "NOT_FOUND", "resource not found")
	case errors.Is(err, platform.ErrConflict):
		writeError(c, http.StatusConflict, "CONFLICT", "resource already exists")
	case errors.Is(err, platform.ErrRateLimited):
		writeError(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
	case errors.Is(err, platform.ErrPayloadTooLarge):
		writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
	case errors.Is(err, platform.ErrUpstream):
		writeError(c, http.StatusBadGateway, "BAD_GATEWAY", "upstream service unavailable")
	default:
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}

func safeMessage(err error, fallback string) string {
	// Prefer a short, user-safe message without wrapping noise.
	msg := err.Error()
	if errors.Is(err, platform.ErrInvalidInput) {
		// Strip the sentinel prefix if present.
		const prefix = "invalid input: "
		if len(msg) > len(prefix) && msg[:len(prefix)] == prefix {
			return msg[len(prefix):]
		}
		if msg == platform.ErrInvalidInput.Error() {
			return fallback
		}
		// fmt.Errorf("%w: detail", ErrInvalidInput) → "invalid input: detail"
		if idx := len("invalid input"); len(msg) > idx+2 && msg[:idx] == "invalid input" {
			return msg[idx+2:]
		}
	}
	return fallback
}

func isMaxBytesError(err error) bool {
	var maxBytes *http.MaxBytesError
	return errors.As(err, &maxBytes)
}
