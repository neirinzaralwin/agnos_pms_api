package domain

import "context"

// Repository is the persistence port for staff. The application layer
// depends on this interface, never on a concrete Postgres type.
type Repository interface {
	Create(ctx context.Context, staff *Staff) error
	FindByUsernameAndHospital(ctx context.Context, username, hospitalCode string) (*Staff, error)
}
