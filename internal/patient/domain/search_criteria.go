package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

// SearchCriteria is a validated patient search request. Hospital is
// deliberately absent — it always comes from the auth context, never from
// user-supplied criteria.
type SearchCriteria struct {
	NationalID  *string
	PassportID  *string
	FirstName   *string
	MiddleName  *string
	LastName    *string
	DateOfBirth *time.Time
	PhoneNumber *string
	Email       *string
	Limit       int
	Offset      int
}

// SearchCriteriaInput is the raw application input before domain parsing.
type SearchCriteriaInput struct {
	NationalID  *string
	PassportID  *string
	FirstName   *string
	MiddleName  *string
	LastName    *string
	DateOfBirth *string // YYYY-MM-DD
	PhoneNumber *string
	Email       *string
	Limit       *int
	Offset      *int
}

// FilterFieldNames returns which filter fields were set (names only, never values).
func (in SearchCriteriaInput) FilterFieldNames() []string {
	names := make([]string, 0, 8)
	addIfSet := func(name string, value *string) {
		if value != nil && strings.TrimSpace(*value) != "" {
			names = append(names, name)
		}
	}
	addIfSet("national_id", in.NationalID)
	addIfSet("passport_id", in.PassportID)
	addIfSet("first_name", in.FirstName)
	addIfSet("middle_name", in.MiddleName)
	addIfSet("last_name", in.LastName)
	addIfSet("date_of_birth", in.DateOfBirth)
	addIfSet("phone_number", in.PhoneNumber)
	addIfSet("email", in.Email)
	return names
}

// ParseSearchCriteria validates that at least one filter is present and
// builds the domain criteria. An empty filter set is rejected rather than
// allowed to fall through to "return everything".
func ParseSearchCriteria(in SearchCriteriaInput) (SearchCriteria, error) {
	if len(in.FilterFieldNames()) == 0 {
		return SearchCriteria{}, fmt.Errorf("%w: at least one search filter is required", apperr.ErrInvalidInput)
	}

	criteria := SearchCriteria{
		NationalID:  trimOptional(in.NationalID),
		PassportID:  trimOptional(in.PassportID),
		FirstName:   trimOptional(in.FirstName),
		MiddleName:  trimOptional(in.MiddleName),
		LastName:    trimOptional(in.LastName),
		PhoneNumber: trimOptional(in.PhoneNumber),
		Email:       trimOptional(in.Email),
	}
	if in.Limit != nil {
		criteria.Limit = *in.Limit
	}
	if in.Offset != nil {
		criteria.Offset = *in.Offset
	}
	if in.DateOfBirth != nil && strings.TrimSpace(*in.DateOfBirth) != "" {
		parsedDOB, operationError := time.Parse("2006-01-02", strings.TrimSpace(*in.DateOfBirth))
		if operationError != nil {
			return SearchCriteria{}, fmt.Errorf("%w: date_of_birth must be YYYY-MM-DD", apperr.ErrInvalidInput)
		}
		criteria.DateOfBirth = &parsedDOB
	}
	return criteria, nil
}

func trimOptional(raw *string) *string {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
