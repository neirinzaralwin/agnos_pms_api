package http

// CreateRequest is the body for POST /staff/create.
type CreateRequest struct {
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

// StaffResponse is returned after create. It never includes the password.
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
