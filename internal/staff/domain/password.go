package domain

import (
	"fmt"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

// Password length policy.
const (
	MinPasswordLength = 12
	MaxPasswordLength = 128
)

// Password is a validated plaintext password. Hashing stays in the
// application layer — this VO only enforces length policy.
type Password struct {
	plaintext string
}

// ParsePassword validates plaintext password policy.
func ParsePassword(raw string) (Password, error) {
	if len(raw) < MinPasswordLength {
		return Password{}, fmt.Errorf("%w: password must be at least %d characters", apperr.ErrInvalidInput, MinPasswordLength)
	}
	if len(raw) > MaxPasswordLength {
		return Password{}, fmt.Errorf("%w: password is too long", apperr.ErrInvalidInput)
	}
	return Password{plaintext: raw}, nil
}

// String returns the plaintext. Callers must not log it.
func (p Password) String() string { return p.plaintext }
