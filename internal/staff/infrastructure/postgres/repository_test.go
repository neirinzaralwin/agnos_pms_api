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
	requestContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, operationError := pgxpool.New(requestContext, dsn)
	require.NoError(t, operationError)
	t.Cleanup(pool.Close)
	require.NoError(t, pool.Ping(requestContext))
	return pool
}

func TestRepository_CreateAndFind(t *testing.T) {
	pool := testPool(t)
	repo := postgres.NewRepository(pool)
	requestContext := context.Background()

	staff := &domain.Staff{Username: "integ-alice", PasswordHash: "hash", Hospital: "hospital-a"}
	require.NoError(t, repo.Create(requestContext, staff))
	require.NotEmpty(t, staff.ID)

	found, operationError := repo.FindByUsernameAndHospital(requestContext, "integ-alice", "hospital-a")
	require.NoError(t, operationError)
	require.Equal(t, staff.ID, found.ID)

	operationError = repo.Create(requestContext, &domain.Staff{Username: "integ-alice", PasswordHash: "hash", Hospital: "hospital-a"})
	require.ErrorIs(t, operationError, platform.ErrConflict)

	_, operationError = repo.FindByUsernameAndHospital(requestContext, "missing", "hospital-a")
	require.ErrorIs(t, operationError, platform.ErrNotFound)
}
