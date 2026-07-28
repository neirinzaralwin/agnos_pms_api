package model

import (
	"fmt"
	"time"
)

// Staff is the staff aggregate root: one person bound to exactly one hospital.
type Staff struct {
	ID           string
	Username     string
	PasswordHash string
	Hospital     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewStaff constructs a staff member ready for persistence.
// passwordHash must already be produced by the application layer (bcrypt).
func NewStaff(username Username, hospital HospitalCode, passwordHash string) (*Staff, error) {
	if username.String() == "" || hospital.IsZero() {
		return nil, fmt.Errorf("%w: username and hospital are required", ErrInvalidInput)
	}
	if passwordHash == "" {
		return nil, fmt.Errorf("%w: password hash is required", ErrInvalidInput)
	}
	return &Staff{
		Username:     username.String(),
		PasswordHash: passwordHash,
		Hospital:     hospital.String(),
	}, nil
}

// HospitalCode returns the staff's hospital as a value object.
func (s *Staff) HospitalCode() HospitalCode {
	h, _ := ParseHospitalCode(s.Hospital)
	return h
}

// BelongsTo reports whether this staff member is bound to hospital.
func (s *Staff) BelongsTo(hospital HospitalCode) bool {
	return s.Hospital == hospital.String()
}
