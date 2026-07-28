// Package hospital holds HospitalCode, the shared-kernel value object both
// the staff and patient contexts scope their aggregates by.
package hospital

import (
	"fmt"
	"strings"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

const maxCodeLength = 64

// Code is a short, normalized hospital identifier (e.g. "hospital-a") — the
// sole authorization scoping key for staff and patients.
type Code struct {
	normalized string
}

// Parse normalizes and validates a raw hospital code.
func Parse(raw string) (Code, error) {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return Code{}, fmt.Errorf("%w: hospital is required", apperr.ErrInvalidInput)
	}
	if len(normalized) > maxCodeLength {
		return Code{}, fmt.Errorf("%w: hospital is too long", apperr.ErrInvalidInput)
	}
	return Code{normalized: normalized}, nil
}

// String returns the normalized code.
func (c Code) String() string { return c.normalized }

// IsZero reports whether the code is unset.
func (c Code) IsZero() bool { return c.normalized == "" }
