package model

import "strings"

// Gender is M, F, or unset. Values outside the HIS contract become unset.
type Gender struct {
	value *string
}

// ParseGender accepts HIS gender values. Unknown or empty → unset (nil), never an error.
func ParseGender(raw *string) Gender {
	if raw == nil {
		return Gender{}
	}
	v := strings.TrimSpace(*raw)
	if v == "M" || v == "F" {
		return Gender{value: &v}
	}
	return Gender{}
}

// Ptr returns the stored value or nil.
func (g Gender) Ptr() *string { return g.value }

// String returns "M", "F", or "".
func (g Gender) String() string {
	if g.value == nil {
		return ""
	}
	return *g.value
}

// IsSet reports whether gender is M or F.
func (g Gender) IsSet() bool { return g.value != nil }
