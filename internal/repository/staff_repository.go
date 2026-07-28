package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
)

// StaffRepository persists staff rows.
type StaffRepository struct {
	pool *pgxpool.Pool
}

// NewStaffRepository constructs a StaffRepository.
func NewStaffRepository(pool *pgxpool.Pool) *StaffRepository {
	return &StaffRepository{pool: pool}
}

// Create inserts a staff member. Returns platform.ErrConflict on unique violation.
func (r *StaffRepository) Create(ctx context.Context, s *model.Staff) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const q = `
		INSERT INTO staff (username, password_hash, hospital)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, q, s.Username, s.PasswordHash, s.Hospital).
		Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return platform.ErrConflict
		}
		return fmt.Errorf("create staff: %w", err)
	}
	return nil
}

// GetByUsernameAndHospital returns the staff row or platform.ErrNotFound.
func (r *StaffRepository) GetByUsernameAndHospital(ctx context.Context, username, hospital string) (*model.Staff, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	const q = `
		SELECT id, username, password_hash, hospital, created_at, updated_at
		FROM staff
		WHERE username = $1 AND hospital = $2
	`
	var s model.Staff
	err := r.pool.QueryRow(ctx, q, username, hospital).Scan(
		&s.ID, &s.Username, &s.PasswordHash, &s.Hospital, &s.CreatedAt, &s.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, platform.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get staff: %w", err)
	}
	return &s, nil
}
