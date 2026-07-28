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
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/domain"
)

type fakeRepository struct {
	CreateFn                    func(requestContext context.Context, staff *domain.Staff) error
	FindByUsernameAndHospitalFn func(requestContext context.Context, username, hospitalCode string) (*domain.Staff, error)
	store                       map[string]*domain.Staff
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{store: make(map[string]*domain.Staff)}
}

func (f *fakeRepository) key(username, hospitalCode string) string {
	return username + "|" + hospitalCode
}

func (f *fakeRepository) Create(requestContext context.Context, staff *domain.Staff) error {
	if f.CreateFn != nil {
		return f.CreateFn(requestContext, staff)
	}
	key := f.key(staff.Username, staff.Hospital)
	if _, exists := f.store[key]; exists {
		return platform.ErrConflict
	}
	staff.ID = "id-" + key
	staff.CreatedAt = time.Now().UTC()
	staff.UpdatedAt = staff.CreatedAt
	stored := *staff
	f.store[key] = &stored
	return nil
}

func (f *fakeRepository) FindByUsernameAndHospital(requestContext context.Context, username, hospitalCode string) (*domain.Staff, error) {
	if f.FindByUsernameAndHospitalFn != nil {
		return f.FindByUsernameAndHospitalFn(requestContext, username, hospitalCode)
	}
	staff, ok := f.store[f.key(username, hospitalCode)]
	if !ok {
		return nil, platform.ErrNotFound
	}
	found := *staff
	return &found, nil
}

func TestCreate_Valid(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	svc := application.NewService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)

	staff, operationError := svc.Create(context.Background(), "alice", "password12345", "Hospital-A")
	require.NoError(t, operationError)
	require.Equal(t, "alice", staff.Username)
	require.Equal(t, "hospital-a", staff.Hospital)
	require.NotEqual(t, "password12345", staff.PasswordHash)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(staff.PasswordHash), []byte("password12345")))
}

func TestCreate_Duplicate(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	svc := application.NewService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)

	_, operationError := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, operationError)
	_, operationError = svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.ErrorIs(t, operationError, platform.ErrConflict)
}

func TestCreate_SameUsernameDifferentHospital(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	svc := application.NewService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)

	_, operationError := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, operationError)
	_, operationError = svc.Create(context.Background(), "alice", "password12345", "hospital-b")
	require.NoError(t, operationError)
}

func TestCreate_ShortPassword(t *testing.T) {
	t.Parallel()
	svc := application.NewService(newFakeRepository(), strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)
	_, operationError := svc.Create(context.Background(), "alice", "short", "hospital-a")
	require.ErrorIs(t, operationError, apperr.ErrInvalidInput)
}

func TestLogin_Valid(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	secret := strings.Repeat("s", 32)
	svc := application.NewService(repo, secret, time.Hour, bcrypt.MinCost, nil)

	_, operationError := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, operationError)

	result, operationError := svc.Login(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, operationError)
	require.NotEmpty(t, result.AccessToken)
	require.Equal(t, int64(3600), result.ExpiresIn)

	claims, operationError := platform.Parse(result.AccessToken, secret)
	require.NoError(t, operationError)
	require.Equal(t, "hospital-a", claims.Hospital)
	require.NotEmpty(t, claims.StaffID)
}

func TestLogin_FailureModesIdentical(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	svc := application.NewService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)
	_, operationError := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, operationError)

	cases := []struct {
		name     string
		username string
		password string
		hospital string
	}{
		{"wrong_password", "alice", "wrong-password-xx", "hospital-a"},
		{"unknown_username", "nobody", "password12345", "hospital-a"},
		{"wrong_hospital", "alice", "password12345", "hospital-b"},
	}

	errs := make([]error, 0, len(cases))
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, operationError := svc.Login(context.Background(), tc.username, tc.password, tc.hospital)
			require.ErrorIs(t, operationError, platform.ErrUnauthorized)
			errs = append(errs, operationError)
		})
	}
	require.Len(t, errs, 3)
	require.Equal(t, errs[0].Error(), errs[1].Error())
	require.Equal(t, errs[1].Error(), errs[2].Error())
}

func TestCreate_RepoError(t *testing.T) {
	t.Parallel()
	repo := newFakeRepository()
	repo.CreateFn = func(requestContext context.Context, staff *domain.Staff) error {
		return context.DeadlineExceeded
	}
	svc := application.NewService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)
	_, operationError := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.Error(t, operationError)
	require.NotErrorIs(t, operationError, platform.ErrConflict)
}
