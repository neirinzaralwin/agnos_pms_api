package hospitala

import "github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"

// wirePatient is the decoded Hospital A HIS response body. It is unexported —
// nothing outside this adapter ever sees Hospital A's wire shape.
type wirePatient struct {
	FirstNameTH  *string `json:"first_name_th"`
	MiddleNameTH *string `json:"middle_name_th"`
	LastNameTH   *string `json:"last_name_th"`
	FirstNameEN  *string `json:"first_name_en"`
	MiddleNameEN *string `json:"middle_name_en"`
	LastNameEN   *string `json:"last_name_en"`
	DateOfBirth  *string `json:"date_of_birth"`
	PatientHN    *string `json:"patient_hn"`
	NationalID   *string `json:"national_id"`
	PassportID   *string `json:"passport_id"`
	PhoneNumber  *string `json:"phone_number"`
	Email        *string `json:"email"`
	Gender       *string `json:"gender"`
}

// toDomain maps the HIS wire shape into the domain-owned snapshot type.
func (w wirePatient) toDomain() *domain.HISPatientData {
	return &domain.HISPatientData{
		FirstNameTH:  w.FirstNameTH,
		MiddleNameTH: w.MiddleNameTH,
		LastNameTH:   w.LastNameTH,
		FirstNameEN:  w.FirstNameEN,
		MiddleNameEN: w.MiddleNameEN,
		LastNameEN:   w.LastNameEN,
		DateOfBirth:  w.DateOfBirth,
		PatientHN:    w.PatientHN,
		NationalID:   w.NationalID,
		PassportID:   w.PassportID,
		PhoneNumber:  w.PhoneNumber,
		Email:        w.Email,
		Gender:       w.Gender,
	}
}
