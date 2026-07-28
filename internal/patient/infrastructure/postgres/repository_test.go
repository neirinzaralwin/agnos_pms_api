//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/infrastructure/postgres"
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

func TestRepository_UpsertAndSearchScoped(t *testing.T) {
	pool := testPool(t)
	repo := postgres.NewRepository(pool)
	ctx := context.Background()

	patient := &domain.Patient{
		Hospital:    "hospital-a",
		FirstNameEN: strPtr("Somchai"),
		NationalID:  strPtr("integ-nid-001"),
		Gender:      strPtr("M"),
	}
	require.NoError(t, repo.Upsert(ctx, patient))
	firstID := patient.ID

	patient.FirstNameEN = strPtr("Updated")
	require.NoError(t, repo.Upsert(ctx, patient))
	require.Equal(t, firstID, patient.ID)

	// Same national id at another hospital is a separate row.
	otherHospitalPatient := &domain.Patient{
		Hospital:    "hospital-b",
		FirstNameEN: strPtr("Somchai"),
		NationalID:  strPtr("integ-nid-001"),
		Gender:      strPtr("M"),
	}
	require.NoError(t, repo.Upsert(ctx, otherHospitalPatient))

	resultsA, err := repo.Search(ctx, "hospital-a", domain.SearchCriteria{NationalID: strPtr("integ-nid-001")})
	require.NoError(t, err)
	require.Len(t, resultsA, 1)
	require.Equal(t, "hospital-a", resultsA[0].Hospital)

	resultsB, err := repo.Search(ctx, "hospital-b", domain.SearchCriteria{NationalID: strPtr("integ-nid-001")})
	require.NoError(t, err)
	require.Len(t, resultsB, 1)
}
