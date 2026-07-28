// Package domain holds the Patient aggregate, its value objects, and the
// ports (Repository, HISClient) the application layer depends on. Nothing
// here imports Gin, SQL drivers, or the HIS transport package.
package domain

import "strings"

// Gender is M, F, or unset. HIS values outside that contract become unset
// rather than an error — the upstream field is informational, not load-bearing.
type Gender struct {
	value *string
}

// ParseGender accepts a raw HIS gender value. Unknown or empty becomes unset.
func ParseGender(raw *string) Gender {
	if raw == nil {
		return Gender{}
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "M" || trimmed == "F" {
		return Gender{value: &trimmed}
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
