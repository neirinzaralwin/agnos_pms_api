package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/application"
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

type fakeRepository struct {
	UpsertFn func(ctx context.Context, patient *domain.Patient) error
	SearchFn func(ctx context.Context, hospitalCode string, criteria domain.SearchCriteria) ([]domain.Patient, error)
	patients []domain.Patient
	upsertN  int
}

func (f *fakeRepository) Upsert(ctx context.Context, patient *domain.Patient) error {
	f.upsertN++
	if f.UpsertFn != nil {
		return f.UpsertFn(ctx, patient)
	}
	if patient.ID == "" {
		patient.ID = "p-1"
	}
	patient.CreatedAt = time.Now().UTC()
	patient.UpdatedAt = patient.CreatedAt
	for i, existing := range f.patients {
		if existing.Hospital != patient.Hospital {
			continue
		}
		sameNationalID := patient.NationalID != nil && existing.NationalID != nil && *patient.NationalID == *existing.NationalID
		samePassportID := patient.PassportID != nil && existing.PassportID != nil && *patient.PassportID == *existing.PassportID
		if sameNationalID || (patient.NationalID == nil && samePassportID) {
			patient.ID = existing.ID
			f.patients[i] = *patient
			return nil
		}
	}
	f.patients = append(f.patients, *patient)
	return nil
}

func (f *fakeRepository) Search(ctx context.Context, hospitalCode string, criteria domain.SearchCriteria) ([]domain.Patient, error) {
	if f.SearchFn != nil {
		return f.SearchFn(ctx, hospitalCode, criteria)
	}
	out := make([]domain.Patient, 0)
	for _, patient := range f.patients {
		if patient.Hospital != hospitalCode {
			continue
		}
		if criteria.NationalID != nil && (patient.NationalID == nil || *patient.NationalID != *criteria.NationalID) {
			continue
		}
		if criteria.PassportID != nil && (patient.PassportID == nil || *patient.PassportID != *criteria.PassportID) {
			continue
		}
		if criteria.FirstName != nil && (patient.FirstNameEN == nil || *patient.FirstNameEN != *criteria.FirstName) {
			continue
		}
		if criteria.Email != nil && (patient.Email == nil || *patient.Email != *criteria.Email) {
			continue
		}
		out = append(out, patient)
	}
	limit := criteria.Limit
	if limit <= 0 {
		limit = 50
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type fakeHISClient struct {
	SearchByIDFn func(ctx context.Context, lookupID string) (*domain.HISPatientData, error)
	called       int
	lastID       string
}

func (f *fakeHISClient) SearchByID(ctx context.Context, lookupID string) (*domain.HISPatientData, error) {
	f.called++
	f.lastID = lookupID
	if f.SearchByIDFn != nil {
		return f.SearchByIDFn(ctx, lookupID)
	}
	return nil, platform.ErrNotFound
}

func strPtr(s string) *string { return &s }

func TestLookupFromHIS_UpsertsWithCallerHospital(t *testing.T) {
	t.Parallel()
	repo := &fakeRepository{}
	his := &fakeHISClient{
		SearchByIDFn: func(ctx context.Context, lookupID string) (*domain.HISPatientData, error) {
			return &domain.HISPatientData{
				FirstNameEN: strPtr("Somchai"),
				LastNameEN:  strPtr("Jaidee"),
				NationalID:  strPtr(lookupID),
				Gender:      strPtr("M"),
				DateOfBirth: strPtr("1990-01-15"),
			}, nil
		},
	}
	svc := application.NewService(repo, his, nil)

	patient, err := svc.LookupFromHIS(context.Background(), "hospital-a", "1100700123456")
	require.NoError(t, err)
	require.Equal(t, "hospital-a", patient.Hospital)
	require.Equal(t, "Somchai", *patient.FirstNameEN)
	require.Equal(t, 1, repo.upsertN)
}

func TestLookupFromHIS_SecondLookupUpdates(t *testing.T) {
	t.Parallel()
	repo := &fakeRepository{}
	callCount := 0
	his := &fakeHISClient{
		SearchByIDFn: func(ctx context.Context, lookupID string) (*domain.HISPatientData, error) {
			callCount++
			name := "First"
			if callCount > 1 {
				name = "Updated"
			}
			return &domain.HISPatientData{
				FirstNameEN: strPtr(name),
				NationalID:  strPtr(lookupID),
				Gender:      strPtr("M"),
			}, nil
		},
	}
	svc := application.NewService(repo, his, nil)

	first, err := svc.LookupFromHIS(context.Background(), "hospital-a", "1100700123456")
	require.NoError(t, err)
	second, err := svc.LookupFromHIS(context.Background(), "hospital-a", "1100700123456")
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, "Updated", *second.FirstNameEN)
	require.Len(t, repo.patients, 1)
}

func TestLookupFromHIS_InvalidID_NeverCallsHIS(t *testing.T) {
	t.Parallel()
	his := &fakeHISClient{}
	svc := application.NewService(&fakeRepository{}, his, nil)

	_, err := svc.LookupFromHIS(context.Background(), "hospital-a", "bad id!")
	require.ErrorIs(t, err, apperr.ErrInvalidInput)
	require.Equal(t, 0, his.called)
}

func TestLookupFromHIS_OddGenderBecomesNil(t *testing.T) {
	t.Parallel()
	repo := &fakeRepository{}
	his := &fakeHISClient{
		SearchByIDFn: func(ctx context.Context, lookupID string) (*domain.HISPatientData, error) {
			return &domain.HISPatientData{
				NationalID: strPtr(lookupID),
				Gender:     strPtr("X"),
			}, nil
		},
	}
	svc := application.NewService(repo, his, nil)
	patient, err := svc.LookupFromHIS(context.Background(), "hospital-a", "ODDGENDER")
	require.NoError(t, err)
	require.Nil(t, patient.Gender)
}

func TestLookupFromHIS_NotFound(t *testing.T) {
	t.Parallel()
	his := &fakeHISClient{SearchByIDFn: func(ctx context.Context, lookupID string) (*domain.HISPatientData, error) {
		return nil, platform.ErrNotFound
	}}
	svc := application.NewService(&fakeRepository{}, his, nil)
	_, err := svc.LookupFromHIS(context.Background(), "hospital-a", "MISSING")
	require.ErrorIs(t, err, platform.ErrNotFound)
}

func TestLookupFromHIS_Upstream(t *testing.T) {
	t.Parallel()
	his := &fakeHISClient{SearchByIDFn: func(ctx context.Context, lookupID string) (*domain.HISPatientData, error) {
		return nil, platform.ErrUpstream
	}}
	svc := application.NewService(&fakeRepository{}, his, nil)
	_, err := svc.LookupFromHIS(context.Background(), "hospital-a", "X")
	require.ErrorIs(t, err, platform.ErrUpstream)
}

func TestSearch_SingleFilter(t *testing.T) {
	t.Parallel()
	repo := &fakeRepository{patients: []domain.Patient{
		{ID: "1", Hospital: "hospital-a", FirstNameEN: strPtr("Somchai"), NationalID: strPtr("1100700123456")},
	}}
	svc := application.NewService(repo, &fakeHISClient{}, nil)

	out, err := svc.Search(context.Background(), "hospital-a", application.SearchFilter{
		NationalID: strPtr("1100700123456"),
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
}

func TestSearch_DoesNotReturnPatientsFromAnotherHospital(t *testing.T) {
	t.Parallel()
	repo := &fakeRepository{patients: []domain.Patient{
		{ID: "1", Hospital: "hospital-b", FirstNameEN: strPtr("Somchai"), NationalID: strPtr("1100700123456"), Email: strPtr("x@y.com")},
	}}
	svc := application.NewService(repo, &fakeHISClient{}, nil)

	out, err := svc.Search(context.Background(), "hospital-a", application.SearchFilter{
		NationalID: strPtr("1100700123456"),
	})
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestSearch_IgnoresBodyHospital(t *testing.T) {
	// Hospital is never taken from the filter — only the JWT hospital argument.
	t.Parallel()
	repo := &fakeRepository{patients: []domain.Patient{
		{ID: "1", Hospital: "hospital-a", Email: strPtr("a@example.com")},
		{ID: "2", Hospital: "hospital-b", Email: strPtr("a@example.com")},
	}}
	svc := application.NewService(repo, &fakeHISClient{}, nil)

	out, err := svc.Search(context.Background(), "hospital-a", application.SearchFilter{
		Email: strPtr("a@example.com"),
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "hospital-a", out[0].Hospital)
}

func TestSearch_NoFilters(t *testing.T) {
	t.Parallel()
	svc := application.NewService(&fakeRepository{}, &fakeHISClient{}, nil)
	_, err := svc.Search(context.Background(), "hospital-a", application.SearchFilter{})
	require.ErrorIs(t, err, apperr.ErrInvalidInput)
}

func TestSearch_EmptySliceNotNil(t *testing.T) {
	t.Parallel()
	svc := application.NewService(&fakeRepository{}, &fakeHISClient{}, nil)
	out, err := svc.Search(context.Background(), "hospital-a", application.SearchFilter{
		Email: strPtr("none@example.com"),
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out, 0)
}

func TestSearch_ResultCap(t *testing.T) {
	t.Parallel()
	patients := make([]domain.Patient, 0, 100)
	for i := 0; i < 100; i++ {
		patients = append(patients, domain.Patient{
			ID:       string(rune('a' + i%26)),
			Hospital: "hospital-a",
			Email:    strPtr("same@example.com"),
		})
	}
	repo := &fakeRepository{patients: patients}
	svc := application.NewService(repo, &fakeHISClient{}, nil)
	limit := 5
	out, err := svc.Search(context.Background(), "hospital-a", application.SearchFilter{
		Email: strPtr("same@example.com"),
		Limit: &limit,
	})
	require.NoError(t, err)
	require.Len(t, out, 5)
}

func TestSearch_SQLMetacharactersLiteral(t *testing.T) {
	t.Parallel()
	evil := "' OR 1=1--"
	repo := &fakeRepository{patients: []domain.Patient{
		{ID: "1", Hospital: "hospital-a", FirstNameEN: strPtr("Normal")},
	}}
	svc := application.NewService(repo, &fakeHISClient{}, nil)
	out, err := svc.Search(context.Background(), "hospital-a", application.SearchFilter{
		FirstName: &evil,
	})
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestSearch_RepoError(t *testing.T) {
	t.Parallel()
	repo := &fakeRepository{
		SearchFn: func(ctx context.Context, hospitalCode string, criteria domain.SearchCriteria) ([]domain.Patient, error) {
			return nil, context.DeadlineExceeded
		},
	}
	svc := application.NewService(repo, &fakeHISClient{}, nil)
	_, err := svc.Search(context.Background(), "hospital-a", application.SearchFilter{
		Email: strPtr("a@b.com"),
	})
	require.Error(t, err)
}
