// Package postgres implements the staff domain.Repository port against
// Postgres via pgx. It is the only place staff SQL is allowed to live.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/domain"
)

// Repository persists staff rows. It implements domain.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a staff Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create inserts a staff member. Returns platform.ErrConflict on unique violation.
func (r *Repository) Create(requestContext context.Context, staff *domain.Staff) error {
	requestContext, cancel := context.WithTimeout(requestContext, 3*time.Second)
	defer cancel()

	const query = `
		INSERT INTO staff (username, password_hash, hospital)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	operationError := r.pool.QueryRow(requestContext, query, staff.Username, staff.PasswordHash, staff.Hospital).
		Scan(&staff.ID, &staff.CreatedAt, &staff.UpdatedAt)
	if operationError != nil {
		var pgErr *pgconn.PgError
		if errors.As(operationError, &pgErr) && pgErr.Code == "23505" {
			return platform.ErrConflict
		}
		return fmt.Errorf("create staff: %w", operationError)
	}
	return nil
}

// FindByUsernameAndHospital returns the staff row or platform.ErrNotFound.
func (r *Repository) FindByUsernameAndHospital(requestContext context.Context, username, hospitalCode string) (*domain.Staff, error) {
	requestContext, cancel := context.WithTimeout(requestContext, 3*time.Second)
	defer cancel()

	const query = `
		SELECT id, username, password_hash, hospital, created_at, updated_at
		FROM staff
		WHERE username = $1 AND hospital = $2
	`
	var staff domain.Staff
	operationError := r.pool.QueryRow(requestContext, query, username, hospitalCode).Scan(
		&staff.ID, &staff.Username, &staff.PasswordHash, &staff.Hospital, &staff.CreatedAt, &staff.UpdatedAt,
	)
	if errors.Is(operationError, pgx.ErrNoRows) {
		return nil, platform.ErrNotFound
	}
	if operationError != nil {
		return nil, fmt.Errorf("get staff: %w", operationError)
	}
	return &staff, nil
}
