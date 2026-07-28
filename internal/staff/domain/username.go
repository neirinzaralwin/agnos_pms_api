package domain

import (
	"fmt"
	"strings"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

// MaxUsernameLength bounds a staff username.
const MaxUsernameLength = 64

// Username is a validated staff username.
type Username struct {
	normalized string
}

// ParseUsername trims and validates a username.
func ParseUsername(raw string) (Username, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Username{}, fmt.Errorf("%w: username is required", apperr.ErrInvalidInput)
	}
	if len(trimmed) > MaxUsernameLength {
		return Username{}, fmt.Errorf("%w: username is too long", apperr.ErrInvalidInput)
	}
	return Username{normalized: trimmed}, nil
}

// String returns the username.
func (u Username) String() string { return u.normalized }
