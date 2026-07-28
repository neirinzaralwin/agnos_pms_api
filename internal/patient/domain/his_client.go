package domain

import "context"

// HISClient is the Hospital A port. The adapter that implements it (the
// anti-corruption layer) is the sole place Hospital A's wire format is
// decoded; this interface only ever exchanges domain types.
type HISClient interface {
	SearchByID(requestContext context.Context, lookupID string) (*HISPatientData, error)
}
