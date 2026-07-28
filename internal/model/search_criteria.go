package model

import (
	"fmt"
	"strings"
	"time"
)

// SearchCriteria is a domain search request. Hospital is never part of criteria —
// it is always supplied separately from the auth context.
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
	add := func(name string, v *string) {
		if v != nil && strings.TrimSpace(*v) != "" {
			names = append(names, name)
		}
	}
	add("national_id", in.NationalID)
	add("passport_id", in.PassportID)
	add("first_name", in.FirstName)
	add("middle_name", in.MiddleName)
	add("last_name", in.LastName)
	add("date_of_birth", in.DateOfBirth)
	add("phone_number", in.PhoneNumber)
	add("email", in.Email)
	return names
}

// ParseSearchCriteria validates that at least one filter is present and builds criteria.
func ParseSearchCriteria(in SearchCriteriaInput) (SearchCriteria, error) {
	if len(in.FilterFieldNames()) == 0 {
		return SearchCriteria{}, fmt.Errorf("%w: at least one search filter is required", ErrInvalidInput)
	}

	c := SearchCriteria{
		NationalID:  trimOptional(in.NationalID),
		PassportID:  trimOptional(in.PassportID),
		FirstName:   trimOptional(in.FirstName),
		MiddleName:  trimOptional(in.MiddleName),
		LastName:    trimOptional(in.LastName),
		PhoneNumber: trimOptional(in.PhoneNumber),
		Email:       trimOptional(in.Email),
	}
	if in.Limit != nil {
		c.Limit = *in.Limit
	}
	if in.Offset != nil {
		c.Offset = *in.Offset
	}
	if in.DateOfBirth != nil && strings.TrimSpace(*in.DateOfBirth) != "" {
		d, err := time.Parse("2006-01-02", strings.TrimSpace(*in.DateOfBirth))
		if err != nil {
			return SearchCriteria{}, fmt.Errorf("%w: date_of_birth must be YYYY-MM-DD", ErrInvalidInput)
		}
		c.DateOfBirth = &d
	}
	return c, nil
}

func trimOptional(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
