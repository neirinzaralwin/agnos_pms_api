//go:build integration

package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/repository"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(ctx))
	return pool
}

func strPtr(s string) *string { return &s }

func TestStaffRepository_CreateAndGet(t *testing.T) {
	pool := testPool(t)
	repo := repository.NewStaffRepository(pool)
	ctx := context.Background()

	s := &model.Staff{Username: "integ-alice", PasswordHash: "hash", Hospital: "hospital-a"}
	require.NoError(t, repo.Create(ctx, s))
	require.NotEmpty(t, s.ID)

	got, err := repo.GetByUsernameAndHospital(ctx, "integ-alice", "hospital-a")
	require.NoError(t, err)
	require.Equal(t, s.ID, got.ID)

	err = repo.Create(ctx, &model.Staff{Username: "integ-alice", PasswordHash: "hash", Hospital: "hospital-a"})
	require.ErrorIs(t, err, platform.ErrConflict)

	_, err = repo.GetByUsernameAndHospital(ctx, "missing", "hospital-a")
	require.ErrorIs(t, err, platform.ErrNotFound)
}

func TestPatientRepository_UpsertAndSearchScoped(t *testing.T) {
	pool := testPool(t)
	repo := repository.NewPatientRepository(pool)
	ctx := context.Background()

	p := &model.Patient{
		Hospital:    "hospital-a",
		FirstNameEN: strPtr("Somchai"),
		NationalID:  strPtr("integ-nid-001"),
		Gender:      strPtr("M"),
	}
	require.NoError(t, repo.Upsert(ctx, p))
	id1 := p.ID

	p.FirstNameEN = strPtr("Updated")
	require.NoError(t, repo.Upsert(ctx, p))
	require.Equal(t, id1, p.ID)

	// Same national id at another hospital is a separate row.
	pB := &model.Patient{
		Hospital:    "hospital-b",
		FirstNameEN: strPtr("Somchai"),
		NationalID:  strPtr("integ-nid-001"),
		Gender:      strPtr("M"),
	}
	require.NoError(t, repo.Upsert(ctx, pB))

	out, err := repo.Search(ctx, "hospital-a", repository.SearchFilter{NationalID: strPtr("integ-nid-001")})
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, "hospital-a", out[0].Hospital)

	outB, err := repo.Search(ctx, "hospital-b", repository.SearchFilter{NationalID: strPtr("integ-nid-001")})
	require.NoError(t, err)
	require.Len(t, outB, 1)
}
