package dto

// CreateStaffRequest is the body for POST /staff/create.
type CreateStaffRequest struct {
	Username string `json:"username" binding:"required,min=1,max=64"`
	Password string `json:"password" binding:"required,min=12,max=128"`
	Hospital string `json:"hospital" binding:"required,min=1,max=64"`
}

// LoginRequest is the body for POST /staff/login.
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=1,max=64"`
	Password string `json:"password" binding:"required,min=1,max=128"`
	Hospital string `json:"hospital" binding:"required,min=1,max=64"`
}

// StaffResponse is returned after create (never includes password).
type StaffResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Hospital  string `json:"hospital"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// LoginResponse is returned after successful login.
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// PatientSearchRequest is the body for POST /patient/search.
// All filter fields are optional but at least one is required (enforced in service).
type PatientSearchRequest struct {
	NationalID  *string `json:"national_id" binding:"omitempty,max=64"`
	PassportID  *string `json:"passport_id" binding:"omitempty,max=64"`
	FirstName   *string `json:"first_name" binding:"omitempty,max=200"`
	MiddleName  *string `json:"middle_name" binding:"omitempty,max=200"`
	LastName    *string `json:"last_name" binding:"omitempty,max=200"`
	DateOfBirth *string `json:"date_of_birth" binding:"omitempty,max=32"`
	PhoneNumber *string `json:"phone_number" binding:"omitempty,max=32"`
	Email       *string `json:"email" binding:"omitempty,max=320"`
	Limit       *int    `json:"limit" binding:"omitempty,min=1,max=200"`
	Offset      *int    `json:"offset" binding:"omitempty,min=0"`
}

// PatientResponse is the JSON shape for a patient record.
type PatientResponse struct {
	ID           string  `json:"id"`
	Hospital     string  `json:"hospital"`
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
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}
