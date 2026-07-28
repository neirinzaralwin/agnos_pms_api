package platform_test

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
)

func TestIssueAndParse(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	now := time.Now().UTC()

	token, err := platform.Issue(platform.TokenClaims{
		StaffID:  "staff-1",
		Hospital: "hospital-a",
	}, secret, time.Hour, now)
	require.NoError(t, err)

	claims, err := platform.Parse(token, secret)
	require.NoError(t, err)
	require.Equal(t, "staff-1", claims.StaffID)
	require.Equal(t, "hospital-a", claims.Hospital)
}

func TestParse_RejectsNoneAlg(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)

	token := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub":      "staff-1",
		"hospital": "hospital-a",
		"exp":      time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = platform.Parse(signed, secret)
	require.Error(t, err)
	require.ErrorIs(t, err, platform.ErrUnauthorized)
}

func TestParse_Expired(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	now := time.Now().Add(-2 * time.Hour)

	token, err := platform.Issue(platform.TokenClaims{
		StaffID:  "staff-1",
		Hospital: "hospital-a",
	}, secret, time.Minute, now)
	require.NoError(t, err)

	_, err = platform.Parse(token, secret)
	require.Error(t, err)
	require.ErrorIs(t, err, platform.ErrUnauthorized)
}
