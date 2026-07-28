package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/hospital"
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/domain"
)

func TestNew(t *testing.T) {
	t.Parallel()
	username, err := domain.ParseUsername("alice")
	require.NoError(t, err)
	code, err := hospital.Parse("hospital-a")
	require.NoError(t, err)

	staff, err := domain.New(username, code, "hash")
	require.NoError(t, err)
	require.True(t, staff.BelongsTo(code))
}
