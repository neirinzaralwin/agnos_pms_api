package application_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/application"
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

func TestSearch_BadDate(t *testing.T) {
	t.Parallel()
	svc := application.NewService(&fakeRepository{}, &fakeHISClient{}, nil)
	_, err := svc.Search(context.Background(), "hospital-a", application.SearchFilter{
		DateOfBirth: strPtr("not-a-date"),
	})
	require.ErrorIs(t, err, apperr.ErrInvalidInput)
}

func TestSearch_ANDFilters(t *testing.T) {
	t.Parallel()
	repo := &fakeRepository{patients: []domain.Patient{
		{ID: "1", Hospital: "hospital-a", FirstNameEN: strPtr("Somchai"), Email: strPtr("a@b.com"), NationalID: strPtr("nid1")},
		{ID: "2", Hospital: "hospital-a", FirstNameEN: strPtr("Somchai"), Email: strPtr("other@b.com"), NationalID: strPtr("nid2")},
	}}
	svc := application.NewService(repo, &fakeHISClient{}, nil)
	out, err := svc.Search(context.Background(), "hospital-a", application.SearchFilter{
		FirstName: strPtr("Somchai"),
		Email:     strPtr("a@b.com"),
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "1", out[0].ID)
}

func TestLookupFromHIS_EmptyHospital(t *testing.T) {
	t.Parallel()
	svc := application.NewService(&fakeRepository{}, &fakeHISClient{}, nil)
	_, err := svc.LookupFromHIS(context.Background(), "", "ABC")
	require.ErrorIs(t, err, apperr.ErrInvalidInput)
}

func TestSearch_EmptyHospital(t *testing.T) {
	t.Parallel()
	svc := application.NewService(&fakeRepository{}, &fakeHISClient{}, nil)
	_, err := svc.Search(context.Background(), "", application.SearchFilter{Email: strPtr("a@b.com")})
	require.ErrorIs(t, err, apperr.ErrInvalidInput)
}
