package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

func TestParseLookupID(t *testing.T) {
	t.Parallel()
	id, operationError := domain.ParseLookupID("A123")
	require.NoError(t, operationError)
	require.Equal(t, "A123", id.String())

	_, operationError = domain.ParseLookupID("bad id!")
	require.ErrorIs(t, operationError, apperr.ErrInvalidInput)
}
