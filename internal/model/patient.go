package model

import "time"

// Patient is a locally stored patient record scoped to one hospital.
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
