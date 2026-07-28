package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/domain"
)

func TestParsePassword(t *testing.T) {
	t.Parallel()
	_, operationError := domain.ParsePassword("short")
	require.ErrorIs(t, operationError, apperr.ErrInvalidInput)

	pwd, operationError := domain.ParsePassword("password12345")
	require.NoError(t, operationError)
	require.Equal(t, "password12345", pwd.String())
}
