package model

import (
	"fmt"
	"strings"
	"time"
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

// HISPatientData is a domain-side snapshot of an upstream HIS patient record.
// The anti-corruption layer (hospitala client + application service) maps HIS
// JSON into this shape so the domain never imports the HIS adapter.
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

// RegisterFromHIS creates a patient aggregate tagged with the caller's hospital
// from an HIS snapshot (upsert identity rules apply in the repository).
func RegisterFromHIS(hospital HospitalCode, data HISPatientData) (*Patient, error) {
	if hospital.IsZero() {
		return nil, fmt.Errorf("%w: hospital required", ErrInvalidInput)
	}

	p := &Patient{
		Hospital:     hospital.String(),
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
		if d, err := time.Parse("2006-01-02", strings.TrimSpace(*data.DateOfBirth)); err == nil {
			p.DateOfBirth = &d
		}
	}
	if !p.HasIdentity() {
		return nil, fmt.Errorf("%w: national_id or passport_id required", ErrInvalidInput)
	}
	return p, nil
}

// HasIdentity reports whether the patient has a national or passport id for upsert.
func (p *Patient) HasIdentity() bool {
	return (p.NationalID != nil && *p.NationalID != "") ||
		(p.PassportID != nil && *p.PassportID != "")
}

// BelongsTo reports whether this patient is scoped to hospital.
func (p *Patient) BelongsTo(hospital HospitalCode) bool {
	return p.Hospital == hospital.String()
}

// HospitalCode returns the patient's hospital as a value object.
func (p *Patient) HospitalCode() HospitalCode {
	h, _ := ParseHospitalCode(p.Hospital)
	return h
}

func emptyToNil(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
