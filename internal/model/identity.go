package model

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	MinPasswordLength = 12
	MaxPasswordLength = 128
	MaxUsernameLength = 64
)

// Password is a validated plaintext password. Hashing stays in the application layer.
type Password struct {
	value string
}

// ParsePassword validates plaintext password policy.
func ParsePassword(raw string) (Password, error) {
	if len(raw) < MinPasswordLength {
		return Password{}, fmt.Errorf("%w: password must be at least %d characters", ErrInvalidInput, MinPasswordLength)
	}
	if len(raw) > MaxPasswordLength {
		return Password{}, fmt.Errorf("%w: password is too long", ErrInvalidInput)
	}
	return Password{value: raw}, nil
}

// String returns the plaintext (callers must not log it).
func (p Password) String() string { return p.value }

// Username is a validated staff username.
type Username struct {
	value string
}

// ParseUsername trims and validates a username.
func ParseUsername(raw string) (Username, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return Username{}, fmt.Errorf("%w: username is required", ErrInvalidInput)
	}
	if len(v) > MaxUsernameLength {
		return Username{}, fmt.Errorf("%w: username is too long", ErrInvalidInput)
	}
	return Username{value: v}, nil
}

// String returns the username.
func (u Username) String() string { return u.value }

// LookupID is a national_id or passport_id used for HIS path lookup.
type LookupID struct {
	value string
}

// ParseLookupID validates alphanumeric, length-bounded HIS path ids.
func ParseLookupID(raw string) (LookupID, error) {
	if len(raw) < 1 || len(raw) > 64 {
		return LookupID{}, fmt.Errorf("%w: invalid id", ErrInvalidInput)
	}
	for _, r := range raw {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return LookupID{}, fmt.Errorf("%w: invalid id", ErrInvalidInput)
		}
	}
	return LookupID{value: raw}, nil
}

// String returns the id.
func (id LookupID) String() string { return id.value }
