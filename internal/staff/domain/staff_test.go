package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/hospital"
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/domain"
)

func TestNew(t *testing.T) {
	t.Parallel()
	username, operationError := domain.ParseUsername("alice")
	require.NoError(t, operationError)
	code, operationError := hospital.Parse("hospital-a")
	require.NoError(t, operationError)

	staff, operationError := domain.New(username, code, "hash")
	require.NoError(t, operationError)
	require.True(t, staff.BelongsTo(code))
}
