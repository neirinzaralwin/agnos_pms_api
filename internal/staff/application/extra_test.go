package application_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/application"
)

type denyLimiter struct{}

func (denyLimiter) Allow(string) bool { return false }

func TestLogin_RateLimited(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	svc := application.NewService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)
	svc.SetLoginLimiter(denyLimiter{})
	_, operationError := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, operationError)
	_, operationError = svc.Login(context.Background(), "alice", "password12345", "hospital-a")
	require.ErrorIs(t, operationError, platform.ErrRateLimited)
}

func TestCreate_EmptyUsername(t *testing.T) {
	t.Parallel()
	svc := application.NewService(newFakeRepository(), strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)
	_, operationError := svc.Create(context.Background(), "  ", "password12345", "hospital-a")
	require.ErrorIs(t, operationError, apperr.ErrInvalidInput)
}
