package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/client/hospitala"
	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/repository"
	"github.com/neirinzaralwin/patient_management_system_api/internal/service"
)

type fakePatientRepo struct {
	UpsertFn func(ctx context.Context, p *model.Patient) error
	SearchFn func(ctx context.Context, hospital string, f repository.SearchFilter) ([]model.Patient, error)
	patients []model.Patient
	upsertN  int
}

func (f *fakePatientRepo) Upsert(ctx context.Context, p *model.Patient) error {
	f.upsertN++
	if f.UpsertFn != nil {
		return f.UpsertFn(ctx, p)
	}
	if p.ID == "" {
		p.ID = "p-1"
	}
	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = p.CreatedAt
	// replace existing by identity
	for i, existing := range f.patients {
		if existing.Hospital == p.Hospital {
			sameNat := p.NationalID != nil && existing.NationalID != nil && *p.NationalID == *existing.NationalID
			samePass := p.PassportID != nil && existing.PassportID != nil && *p.PassportID == *existing.PassportID
			if sameNat || (p.NationalID == nil && samePass) {
				p.ID = existing.ID
				f.patients[i] = *p
				return nil
			}
		}
	}
	f.patients = append(f.patients, *p)
	return nil
}

func (f *fakePatientRepo) Search(ctx context.Context, hospital string, filter repository.SearchFilter) ([]model.Patient, error) {
	if f.SearchFn != nil {
		return f.SearchFn(ctx, hospital, filter)
	}
	out := make([]model.Patient, 0)
	for _, p := range f.patients {
		if p.Hospital != hospital {
			continue
		}
		if filter.NationalID != nil && (p.NationalID == nil || *p.NationalID != *filter.NationalID) {
			continue
		}
		if filter.PassportID != nil && (p.PassportID == nil || *p.PassportID != *filter.PassportID) {
			continue
		}
		if filter.FirstName != nil && (p.FirstNameEN == nil || *p.FirstNameEN != *filter.FirstName) {
			continue
		}
		if filter.Email != nil && (p.Email == nil || *p.Email != *filter.Email) {
			continue
		}
		out = append(out, p)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type fakeHIS struct {
	SearchByIDFn func(ctx context.Context, id string) (*hospitala.Patient, error)
	called       int
	lastID       string
}

func (f *fakeHIS) SearchByID(ctx context.Context, id string) (*hospitala.Patient, error) {
	f.called++
	f.lastID = id
	if f.SearchByIDFn != nil {
		return f.SearchByIDFn(ctx, id)
	}
	return nil, platform.ErrNotFound
}

func strPtr(s string) *string { return &s }

func TestLookupFromHIS_UpsertsWithCallerHospital(t *testing.T) {
	t.Parallel()
	repo := &fakePatientRepo{}
	his := &fakeHIS{
		SearchByIDFn: func(ctx context.Context, id string) (*hospitala.Patient, error) {
			return &hospitala.Patient{
				FirstNameEN: strPtr("Somchai"),
				LastNameEN:  strPtr("Jaidee"),
				NationalID:  strPtr(id),
				Gender:      strPtr("M"),
				DateOfBirth: strPtr("1990-01-15"),
			}, nil
		},
	}
	svc := service.NewPatientService(repo, his, nil)

	p, err := svc.LookupFromHIS(context.Background(), "hospital-a", "1100700123456")
	require.NoError(t, err)
	require.Equal(t, "hospital-a", p.Hospital)
	require.Equal(t, "Somchai", *p.FirstNameEN)
	require.Equal(t, 1, repo.upsertN)
}

func TestLookupFromHIS_SecondLookupUpdates(t *testing.T) {
	t.Parallel()
	repo := &fakePatientRepo{}
	n := 0
	his := &fakeHIS{
		SearchByIDFn: func(ctx context.Context, id string) (*hospitala.Patient, error) {
			n++
			name := "First"
			if n > 1 {
				name = "Updated"
			}
			return &hospitala.Patient{
				FirstNameEN: strPtr(name),
				NationalID:  strPtr(id),
				Gender:      strPtr("M"),
			}, nil
		},
	}
	svc := service.NewPatientService(repo, his, nil)

	p1, err := svc.LookupFromHIS(context.Background(), "hospital-a", "1100700123456")
	require.NoError(t, err)
	p2, err := svc.LookupFromHIS(context.Background(), "hospital-a", "1100700123456")
	require.NoError(t, err)
	require.Equal(t, p1.ID, p2.ID)
	require.Equal(t, "Updated", *p2.FirstNameEN)
	require.Len(t, repo.patients, 1)
}

func TestLookupFromHIS_InvalidID_NeverCallsHIS(t *testing.T) {
	t.Parallel()
	his := &fakeHIS{}
	svc := service.NewPatientService(&fakePatientRepo{}, his, nil)

	_, err := svc.LookupFromHIS(context.Background(), "hospital-a", "bad id!")
	require.ErrorIs(t, err, model.ErrInvalidInput)
	require.Equal(t, 0, his.called)
}

func TestLookupFromHIS_OddGenderBecomesNil(t *testing.T) {
	t.Parallel()
	repo := &fakePatientRepo{}
	his := &fakeHIS{
		SearchByIDFn: func(ctx context.Context, id string) (*hospitala.Patient, error) {
			return &hospitala.Patient{
				NationalID: strPtr(id),
				Gender:     strPtr("X"),
			}, nil
		},
	}
	svc := service.NewPatientService(repo, his, nil)
	p, err := svc.LookupFromHIS(context.Background(), "hospital-a", "ODDGENDER")
	require.NoError(t, err)
	require.Nil(t, p.Gender)
}

func TestLookupFromHIS_NotFound(t *testing.T) {
	t.Parallel()
	his := &fakeHIS{SearchByIDFn: func(ctx context.Context, id string) (*hospitala.Patient, error) {
		return nil, platform.ErrNotFound
	}}
	svc := service.NewPatientService(&fakePatientRepo{}, his, nil)
	_, err := svc.LookupFromHIS(context.Background(), "hospital-a", "MISSING")
	require.ErrorIs(t, err, platform.ErrNotFound)
}

func TestLookupFromHIS_Upstream(t *testing.T) {
	t.Parallel()
	his := &fakeHIS{SearchByIDFn: func(ctx context.Context, id string) (*hospitala.Patient, error) {
		return nil, platform.ErrUpstream
	}}
	svc := service.NewPatientService(&fakePatientRepo{}, his, nil)
	_, err := svc.LookupFromHIS(context.Background(), "hospital-a", "X")
	require.ErrorIs(t, err, platform.ErrUpstream)
}

func TestPatientSearch_SingleFilter(t *testing.T) {
	t.Parallel()
	repo := &fakePatientRepo{patients: []model.Patient{
		{ID: "1", Hospital: "hospital-a", FirstNameEN: strPtr("Somchai"), NationalID: strPtr("1100700123456")},
	}}
	svc := service.NewPatientService(repo, &fakeHIS{}, nil)

	out, err := svc.Search(context.Background(), "hospital-a", service.SearchFilter{
		NationalID: strPtr("1100700123456"),
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
}

func TestPatientSearch_DoesNotReturnPatientsFromAnotherHospital(t *testing.T) {
	t.Parallel()
	repo := &fakePatientRepo{patients: []model.Patient{
		{ID: "1", Hospital: "hospital-b", FirstNameEN: strPtr("Somchai"), NationalID: strPtr("1100700123456"), Email: strPtr("x@y.com")},
	}}
	svc := service.NewPatientService(repo, &fakeHIS{}, nil)

	out, err := svc.Search(context.Background(), "hospital-a", service.SearchFilter{
		NationalID: strPtr("1100700123456"),
	})
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestPatientSearch_IgnoresBodyHospital(t *testing.T) {
	// Hospital is never taken from the filter — only the JWT hospital arg.
	t.Parallel()
	repo := &fakePatientRepo{patients: []model.Patient{
		{ID: "1", Hospital: "hospital-a", Email: strPtr("a@example.com")},
		{ID: "2", Hospital: "hospital-b", Email: strPtr("a@example.com")},
	}}
	svc := service.NewPatientService(repo, &fakeHIS{}, nil)

	out, err := svc.Search(context.Background(), "hospital-a", service.SearchFilter{
		Email: strPtr("a@example.com"),
	})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "hospital-a", out[0].Hospital)
}

func TestPatientSearch_NoFilters(t *testing.T) {
	t.Parallel()
	svc := service.NewPatientService(&fakePatientRepo{}, &fakeHIS{}, nil)
	_, err := svc.Search(context.Background(), "hospital-a", service.SearchFilter{})
	require.ErrorIs(t, err, model.ErrInvalidInput)
}

func TestPatientSearch_EmptySliceNotNil(t *testing.T) {
	t.Parallel()
	svc := service.NewPatientService(&fakePatientRepo{}, &fakeHIS{}, nil)
	out, err := svc.Search(context.Background(), "hospital-a", service.SearchFilter{
		Email: strPtr("none@example.com"),
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Len(t, out, 0)
}

func TestPatientSearch_ResultCap(t *testing.T) {
	t.Parallel()
	patients := make([]model.Patient, 0, 100)
	for i := 0; i < 100; i++ {
		patients = append(patients, model.Patient{
			ID:       string(rune('a' + i%26)),
			Hospital: "hospital-a",
			Email:    strPtr("same@example.com"),
		})
	}
	repo := &fakePatientRepo{patients: patients}
	svc := service.NewPatientService(repo, &fakeHIS{}, nil)
	limit := 5
	out, err := svc.Search(context.Background(), "hospital-a", service.SearchFilter{
		Email: strPtr("same@example.com"),
		Limit: &limit,
	})
	require.NoError(t, err)
	require.Len(t, out, 5)
}

func TestPatientSearch_SQLMetacharactersLiteral(t *testing.T) {
	t.Parallel()
	evil := "' OR 1=1--"
	repo := &fakePatientRepo{patients: []model.Patient{
		{ID: "1", Hospital: "hospital-a", FirstNameEN: strPtr("Normal")},
	}}
	svc := service.NewPatientService(repo, &fakeHIS{}, nil)
	out, err := svc.Search(context.Background(), "hospital-a", service.SearchFilter{
		FirstName: &evil,
	})
	require.NoError(t, err)
	require.Empty(t, out)
}

func TestPatientSearch_RepoError(t *testing.T) {
	t.Parallel()
	repo := &fakePatientRepo{
		SearchFn: func(ctx context.Context, hospital string, f repository.SearchFilter) ([]model.Patient, error) {
			return nil, context.DeadlineExceeded
		},
	}
	svc := service.NewPatientService(repo, &fakeHIS{}, nil)
	_, err := svc.Search(context.Background(), "hospital-a", service.SearchFilter{
		Email: strPtr("a@b.com"),
	})
	require.Error(t, err)
}
