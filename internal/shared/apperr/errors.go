// Package apperr holds the domain-level validation sentinel shared by every
// bounded context. Infrastructure-level failures (not found, conflict,
// unauthorized, upstream, rate limited) live in platform instead.
package apperr

import "errors"

// ErrInvalidInput marks a value-object or aggregate validation failure.
// Transport maps it to HTTP 400 INVALID_INPUT.
var ErrInvalidInput = errors.New("invalid input")
