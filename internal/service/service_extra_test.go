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

type denyLimiter struct{}

func (denyLimiter) Allow(string) bool { return false }

func TestStaffLogin_RateLimited(t *testing.T) {
	t.Parallel()
	repo := newFakeStaffRepo()
	svc := service.NewStaffService(repo, strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)
	svc.SetLoginLimiter(denyLimiter{})
	_, err := svc.Create(context.Background(), "alice", "password12345", "hospital-a")
	require.NoError(t, err)
	_, err = svc.Login(context.Background(), "alice", "password12345", "hospital-a")
	require.ErrorIs(t, err, platform.ErrRateLimited)
}

func TestStaffCreate_EmptyUsername(t *testing.T) {
	t.Parallel()
	svc := service.NewStaffService(newFakeStaffRepo(), strings.Repeat("s", 32), time.Hour, bcrypt.MinCost, nil)
	_, err := svc.Create(context.Background(), "  ", "password12345", "hospital-a")
	require.ErrorIs(t, err, model.ErrInvalidInput)
}

func TestPatientSearch_BadDate(t *testing.T) {
	t.Parallel()
	svc := service.NewPatientService(&fakePatientRepo{}, &fakeHIS{}, nil)
	_, err := svc.Search(context.Background(), "hospital-a", service.SearchFilter{
		DateOfBirth: strPtr("not-a-date"),
	})
	require.ErrorIs(t, err, model.ErrInvalidInput)
}

func TestPatientSearch_ANDFilters(t *testing.T) {
	t.Parallel()
	repo := &fakePatientRepo{patients: []model.Patient{
		{ID: "1", Hospital: "hospital-a", FirstNameEN: strPtr("Somchai"), Email: strPtr("a@b.com"), NationalID: strPtr("nid1")},
		{ID: "2", Hospital: "hospital-a", FirstNameEN: strPtr("Somchai"), Email: strPtr("other@b.com"), NationalID: strPtr("nid2")},
	}}
	svc := service.NewPatientService(repo, &fakeHIS{}, nil)
	out, err := svc.Search(context.Background(), "hospital-a", service.SearchFilter{
		FirstName: strPtr("Somchai"),
		Email:     strPtr("a@b.com"),
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "1", out[0].ID)
}

func TestLookupFromHIS_EmptyHospital(t *testing.T) {
	t.Parallel()
	svc := service.NewPatientService(&fakePatientRepo{}, &fakeHIS{}, nil)
	_, err := svc.LookupFromHIS(context.Background(), "", "ABC")
	require.ErrorIs(t, err, model.ErrInvalidInput)
}

func TestPatientSearch_EmptyHospital(t *testing.T) {
	t.Parallel()
	svc := service.NewPatientService(&fakePatientRepo{}, &fakeHIS{}, nil)
	_, err := svc.Search(context.Background(), "", service.SearchFilter{Email: strPtr("a@b.com")})
	require.ErrorIs(t, err, model.ErrInvalidInput)
}
