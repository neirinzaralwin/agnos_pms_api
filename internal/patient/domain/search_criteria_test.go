package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
)

func TestParseSearchCriteria_RequiresFilter(t *testing.T) {
	t.Parallel()
	_, err := domain.ParseSearchCriteria(domain.SearchCriteriaInput{})
	require.ErrorIs(t, err, apperr.ErrInvalidInput)

	email := "a@b.com"
	criteria, err := domain.ParseSearchCriteria(domain.SearchCriteriaInput{Email: &email})
	require.NoError(t, err)
	require.Equal(t, "a@b.com", *criteria.Email)
}
