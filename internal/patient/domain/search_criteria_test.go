package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

func TestParseSearchCriteria_RequiresFilter(t *testing.T) {
	t.Parallel()
	_, operationError := domain.ParseSearchCriteria(domain.SearchCriteriaInput{})
	require.ErrorIs(t, operationError, apperr.ErrInvalidInput)

	email := "a@b.com"
	criteria, operationError := domain.ParseSearchCriteria(domain.SearchCriteriaInput{Email: &email})
	require.NoError(t, operationError)
	require.Equal(t, "a@b.com", *criteria.Email)
}
