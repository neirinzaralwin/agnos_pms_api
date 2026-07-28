// Package domain holds the Staff aggregate and its value objects. Nothing
// here imports Gin, SQL drivers, or the HIS client — only stdlib and the
// shared kernel.
package domain

import (
	"fmt"
	"time"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/hospital"
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

// New constructs a staff member ready for persistence. passwordHash must
// already be produced by the application layer (bcrypt) — the aggregate
// never hashes passwords itself.
func New(username Username, hospitalCode hospital.Code, passwordHash string) (*Staff, error) {
	if username.String() == "" || hospitalCode.IsZero() {
		return nil, fmt.Errorf("%w: username and hospital are required", apperr.ErrInvalidInput)
	}
	if passwordHash == "" {
		return nil, fmt.Errorf("%w: password hash is required", apperr.ErrInvalidInput)
	}
	return &Staff{
		Username:     username.String(),
		PasswordHash: passwordHash,
		Hospital:     hospitalCode.String(),
	}, nil
}

// HospitalCode returns the staff's hospital as a value object.
func (s *Staff) HospitalCode() hospital.Code {
	code, _ := hospital.Parse(s.Hospital)
	return code
}

// BelongsTo reports whether this staff member is bound to hospitalCode.
func (s *Staff) BelongsTo(hospitalCode hospital.Code) bool {
	return s.Hospital == hospitalCode.String()
}
