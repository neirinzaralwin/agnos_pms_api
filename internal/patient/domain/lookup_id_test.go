package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
)

func TestParseLookupID(t *testing.T) {
	t.Parallel()
	id, err := domain.ParseLookupID("A123")
	require.NoError(t, err)
	require.Equal(t, "A123", id.String())

	_, err = domain.ParseLookupID("bad id!")
	require.ErrorIs(t, err, apperr.ErrInvalidInput)
}
