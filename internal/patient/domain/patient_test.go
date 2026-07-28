package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/hospital"
)

func TestRegisterFromHIS(t *testing.T) {
	t.Parallel()
	code, operationError := hospital.Parse("hospital-a")
	require.NoError(t, operationError)

	nationalID := "1100700123456"
	oddGender := "X"
	name := "Somchai"
	patient, operationError := domain.RegisterFromHIS(code, domain.HISPatientData{
		FirstNameEN: &name,
		NationalID:  &nationalID,
		Gender:      &oddGender,
	})
	require.NoError(t, operationError)
	require.True(t, patient.BelongsTo(code))
	require.Nil(t, patient.Gender) // odd gender coerced to unset
	require.True(t, patient.HasIdentity())
}
