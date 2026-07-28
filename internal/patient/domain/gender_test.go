package domain_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
)

func TestParseGender(t *testing.T) {
	t.Parallel()
	male := "M"
	unknown := "X"
	require.Equal(t, "M", domain.ParseGender(&male).String())
	require.False(t, domain.ParseGender(&unknown).IsSet())
	require.False(t, domain.ParseGender(nil).IsSet())
}
