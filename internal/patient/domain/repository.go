package domain

import "context"

// Repository is the persistence port for patients. Every method takes
// hospitalCode as a mandatory argument — there is no method capable of
// returning patients across hospitals.
type Repository interface {
	Upsert(requestContext context.Context, patient *Patient) error
	Search(requestContext context.Context, hospitalCode string, criteria SearchCriteria) ([]Patient, error)
}
