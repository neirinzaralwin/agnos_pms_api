//go:build integration

package postgres_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/infrastructure/postgres"
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

func TestRepository_CreateAndFind(t *testing.T) {
	pool := testPool(t)
	repo := postgres.NewRepository(pool)
	ctx := context.Background()

	staff := &domain.Staff{Username: "integ-alice", PasswordHash: "hash", Hospital: "hospital-a"}
	require.NoError(t, repo.Create(ctx, staff))
	require.NotEmpty(t, staff.ID)

	found, err := repo.FindByUsernameAndHospital(ctx, "integ-alice", "hospital-a")
	require.NoError(t, err)
	require.Equal(t, staff.ID, found.ID)

	err = repo.Create(ctx, &domain.Staff{Username: "integ-alice", PasswordHash: "hash", Hospital: "hospital-a"})
	require.ErrorIs(t, err, platform.ErrConflict)

	_, err = repo.FindByUsernameAndHospital(ctx, "missing", "hospital-a")
	require.ErrorIs(t, err, platform.ErrNotFound)
}
