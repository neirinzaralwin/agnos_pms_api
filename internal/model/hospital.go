package model

import (
	"fmt"
	"strings"
)

// HospitalCode is a short normalized hospital identifier (e.g. "hospital-a").
// It is the authorization scoping key for staff and patients.
type HospitalCode struct {
	value string
}

// ParseHospitalCode normalizes and validates a hospital code.
func ParseHospitalCode(raw string) (HospitalCode, error) {
	v := strings.ToLower(strings.TrimSpace(raw))
	if v == "" {
		return HospitalCode{}, fmt.Errorf("%w: hospital is required", ErrInvalidInput)
	}
	if len(v) > 64 {
		return HospitalCode{}, fmt.Errorf("%w: hospital is too long", ErrInvalidInput)
	}
	return HospitalCode{value: v}, nil
}

// String returns the normalized code.
func (h HospitalCode) String() string { return h.value }

// IsZero reports whether the code is empty.
func (h HospitalCode) IsZero() bool { return h.value == "" }
