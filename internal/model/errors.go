package model

import "errors"

// Domain validation sentinel. Handlers map this to HTTP 400 INVALID_INPUT.
var ErrInvalidInput = errors.New("invalid input")
