package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/hospital"
)

func TestRegisterFromHIS(t *testing.T) {
	t.Parallel()
	code, err := hospital.Parse("hospital-a")
	require.NoError(t, err)

	nationalID := "1100700123456"
	oddGender := "X"
	name := "Somchai"
	patient, err := domain.RegisterFromHIS(code, domain.HISPatientData{
		FirstNameEN: &name,
		NationalID:  &nationalID,
		Gender:      &oddGender,
	})
	require.NoError(t, err)
	require.True(t, patient.BelongsTo(code))
	require.Nil(t, patient.Gender) // odd gender coerced to unset
	require.True(t, patient.HasIdentity())
}
