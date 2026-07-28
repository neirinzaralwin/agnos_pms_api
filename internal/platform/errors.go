package platform

import "errors"

// Sentinel errors for expected domain conditions. Handlers map these to HTTP.
var (
	ErrNotFound        = errors.New("not found")
	ErrConflict        = errors.New("conflict")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrInvalidInput    = errors.New("invalid input")
	ErrUpstream        = errors.New("upstream failure")
	ErrRateLimited     = errors.New("rate limited")
	ErrPayloadTooLarge = errors.New("payload too large")
)
