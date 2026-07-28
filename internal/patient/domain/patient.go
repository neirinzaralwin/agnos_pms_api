package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/hospital"
)

// Patient is the patient aggregate root, always scoped to one hospital.
type Patient struct {
	ID           string
	Hospital     string
	FirstNameTH  *string
	MiddleNameTH *string
	LastNameTH   *string
	FirstNameEN  *string
	MiddleNameEN *string
	LastNameEN   *string
	DateOfBirth  *time.Time // calendar date; time component ignored
	PatientHN    *string
	NationalID   *string
	PassportID   *string
	PhoneNumber  *string
	Email        *string
	Gender       *string // "M", "F", or nil
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// HISPatientData is the domain-side shape of an upstream HIS patient record.
// The hospitala adapter (the anti-corruption layer) maps its own wire format
// into this shape, so the domain and application layers never see HIS JSON.
type HISPatientData struct {
	FirstNameTH  *string
	MiddleNameTH *string
	LastNameTH   *string
	FirstNameEN  *string
	MiddleNameEN *string
	LastNameEN   *string
	DateOfBirth  *string // YYYY-MM-DD
	PatientHN    *string
	NationalID   *string
	PassportID   *string
	PhoneNumber  *string
	Email        *string
	Gender       *string
}

// RegisterFromHIS builds a patient aggregate tagged with the caller's
// hospital from an HIS snapshot. Upsert identity rules are enforced by the
// repository, not here.
func RegisterFromHIS(hospitalCode hospital.Code, data HISPatientData) (*Patient, error) {
	if hospitalCode.IsZero() {
		return nil, fmt.Errorf("%w: hospital required", apperr.ErrInvalidInput)
	}

	patient := &Patient{
		Hospital:     hospitalCode.String(),
		FirstNameTH:  emptyToNil(data.FirstNameTH),
		MiddleNameTH: emptyToNil(data.MiddleNameTH),
		LastNameTH:   emptyToNil(data.LastNameTH),
		FirstNameEN:  emptyToNil(data.FirstNameEN),
		MiddleNameEN: emptyToNil(data.MiddleNameEN),
		LastNameEN:   emptyToNil(data.LastNameEN),
		PatientHN:    emptyToNil(data.PatientHN),
		NationalID:   emptyToNil(data.NationalID),
		PassportID:   emptyToNil(data.PassportID),
		PhoneNumber:  emptyToNil(data.PhoneNumber),
		Email:        emptyToNil(data.Email),
		Gender:       ParseGender(data.Gender).Ptr(),
	}
	if data.DateOfBirth != nil && strings.TrimSpace(*data.DateOfBirth) != "" {
		if parsedDOB, err := time.Parse("2006-01-02", strings.TrimSpace(*data.DateOfBirth)); err == nil {
			patient.DateOfBirth = &parsedDOB
		}
	}
	if !patient.HasIdentity() {
		return nil, fmt.Errorf("%w: national_id or passport_id required", apperr.ErrInvalidInput)
	}
	return patient, nil
}

// HasIdentity reports whether the patient has a national or passport id, the
// precondition for an upsert.
func (p *Patient) HasIdentity() bool {
	return (p.NationalID != nil && *p.NationalID != "") ||
		(p.PassportID != nil && *p.PassportID != "")
}

// BelongsTo reports whether this patient is scoped to hospitalCode.
func (p *Patient) BelongsTo(hospitalCode hospital.Code) bool {
	return p.Hospital == hospitalCode.String()
}

// HospitalCode returns the patient's hospital as a value object.
func (p *Patient) HospitalCode() hospital.Code {
	code, _ := hospital.Parse(p.Hospital)
	return code
}

func emptyToNil(raw *string) *string {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
