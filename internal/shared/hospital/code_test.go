package hospital_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/hospital"
)

func TestParse(t *testing.T) {
	t.Parallel()
	code, err := hospital.Parse("  Hospital-A ")
	require.NoError(t, err)
	require.Equal(t, "hospital-a", code.String())

	_, err = hospital.Parse("  ")
	require.ErrorIs(t, err, apperr.ErrInvalidInput)
}
