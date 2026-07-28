package model

import "time"

// Staff is a hospital staff member with login credentials.
type Staff struct {
	ID           string
	Username     string
	PasswordHash string
	Hospital     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
