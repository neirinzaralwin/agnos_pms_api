package platform

import "errors"

// Sentinel errors for infrastructure/transport-level failure modes. Handlers
// map these to HTTP. Domain-level validation failures use apperr.ErrInvalidInput
// instead — see internal/shared/apperr.
var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrUpstream        = errors.New("upstream failure")
	ErrRateLimited     = errors.New("rate limited")
	ErrPayloadTooLarge = errors.New("payload too large")
)
