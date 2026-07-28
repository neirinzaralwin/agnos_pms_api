package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/service"
)

type fakeStaffRepo struct {
	CreateFn                   func(ctx context.Context, s *model.Staff) error
	GetByUsernameAndHospitalFn func(ctx context.Context, username, hospital string) (*model.Staff, error)
	store                      map[string]*model.Staff
}

func newFakeStaffRepo() *fakeStaffRepo {
	return &fakeStaffRepo{store: make(map[string]*model.Staff)}
}

func (f *fakeStaffRepo) key(u, h string) string { return u + "|" + h }

func (f *fakeStaffRepo) Create(ctx context.Context, s *model.Staff) error {
	if f.CreateFn != nil {
		return f.CreateFn(ctx, s)
	}
	k := f.key(s.Username, s.Hospital)
	if _, ok := f.store[k]; ok {
		return platform.ErrConflict
	}
	s.ID = "id-" + k
	s.CreatedAt = time.Now().UTC()
	s.UpdatedAt = s.CreatedAt
	cp := *s
	f.store[k] = &cp
	return nil
}

func (f *fakeStaffRepo) GetByUsernameAndHospital(ctx context.Context, username, hospital string) (*model.Staff, error) {
	if f.GetByUsernameAndHospitalFn != nil {
		return f.GetByUsernameAndHospitalFn(ctx, username, hospital)
	}
	s, ok := f.store[f.key(username, hospital)]
	if !ok {
		return nil, platform.ErrNotFound
	}
	cp := *s
	return &cp, nil
}

func TestStaffCreate_Valid(t *testing.T) {
	t.Parallel()
	repo := newFakeStaffRepo()
	svc := service.NewStaffService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)

	staff, err := svc.Create(context.Background(), "alice", "password12345", "Hospital-A")
	require.NoError(t, err)
	require.Equal(t, "alice", staff.Username)
	require.Equal(t, "hospital-a", staff.Hospital)
	require.NotEqual(t, "password12345", staff.PasswordHash)
	require.NoError(t, bcrypt.CompareHashAndPassword([]byte(staff.PasswordHash), []byte("password12345")))
}

func TestStaffCreate_Duplicate(t *testing.T) {
	t.Parallel()
	repo := newFakeStaffRepo()
	svc := service.NewStaffService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)

	_, err := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.ErrorIs(t, err, platform.ErrConflict)
}

func TestStaffCreate_SameUsernameDifferentHospital(t *testing.T) {
	t.Parallel()
	repo := newFakeStaffRepo()
	svc := service.NewStaffService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)

	_, err := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, err)
	_, err = svc.Create(context.Background(), "alice", "password12345", "hospital-b")
	require.NoError(t, err)
}

func TestStaffCreate_ShortPassword(t *testing.T) {
	t.Parallel()
	svc := service.NewStaffService(newFakeStaffRepo(), strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)
	_, err := svc.Create(context.Background(), "alice", "short", "hospital-a")
	require.ErrorIs(t, err, platform.ErrInvalidInput)
}

func TestStaffLogin_Valid(t *testing.T) {
	t.Parallel()
	repo := newFakeStaffRepo()
	secret := strings.Repeat("s", 32)
	svc := service.NewStaffService(repo, secret, time.Hour, bcrypt.MinCost, nil)

	_, err := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, err)

	result, err := svc.Login(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, err)
	require.NotEmpty(t, result.AccessToken)
	require.Equal(t, int64(3600), result.ExpiresIn)

	claims, err := platform.Parse(result.AccessToken, secret)
	require.NoError(t, err)
	require.Equal(t, "hospital-a", claims.Hospital)
	require.NotEmpty(t, claims.StaffID)
}

func TestStaffLogin_FailureModesIdentical(t *testing.T) {
	t.Parallel()
	repo := newFakeStaffRepo()
	svc := service.NewStaffService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)
	_, err := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, err)

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
			_, err := svc.Login(context.Background(), tc.username, tc.password, tc.hospital)
			require.ErrorIs(t, err, platform.ErrUnauthorized)
			errs = append(errs, err)
		})
	}
	require.Len(t, errs, 3)
	require.Equal(t, errs[0].Error(), errs[1].Error())
	require.Equal(t, errs[1].Error(), errs[2].Error())
}

func TestStaffCreate_RepoError(t *testing.T) {
	t.Parallel()
	repo := newFakeStaffRepo()
	repo.CreateFn = func(ctx context.Context, s *model.Staff) error {
		return context.DeadlineExceeded
	}
	svc := service.NewStaffService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)
	_, err := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.Error(t, err)
	require.NotErrorIs(t, err, platform.ErrConflict)
}
