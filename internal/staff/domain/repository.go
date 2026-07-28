package domain

import "context"

// Repository is the persistence port for staff. The application layer
// depends on this interface, never on a concrete Postgres type.
type Repository interface {
	Create(requestContext context.Context, staff *Staff) error
	FindByUsernameAndHospital(requestContext context.Context, username, hospitalCode string) (*Staff, error)
}
