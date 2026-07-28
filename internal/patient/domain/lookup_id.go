package domain

import (
	"fmt"
	"unicode"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

// LookupID is a national_id or passport_id used for the HIS path lookup.
type LookupID struct {
	value string
}

// ParseLookupID validates an alphanumeric, length-bounded HIS path id.
func ParseLookupID(raw string) (LookupID, error) {
	if len(raw) < 1 || len(raw) > 64 {
		return LookupID{}, fmt.Errorf("%w: invalid id", apperr.ErrInvalidInput)
	}
	for _, ch := range raw {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) {
			return LookupID{}, fmt.Errorf("%w: invalid id", apperr.ErrInvalidInput)
		}
	}
	return LookupID{value: raw}, nil
}

// String returns the id.
func (id LookupID) String() string { return id.value }
