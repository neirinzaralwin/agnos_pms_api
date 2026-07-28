// Package postgres implements the patient domain.Repository port against
// Postgres via pgx. It is the only place patient SQL is allowed to live.
package postgres

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

// Repository persists patient rows. It implements domain.Repository.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a patient Repository.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const patientColumns = `
	id, hospital,
	first_name_th, middle_name_th, last_name_th,
	first_name_en, middle_name_en, last_name_en,
	date_of_birth, patient_hn, national_id, passport_id,
	phone_number, email, gender, created_at, updated_at
`

// Upsert inserts or updates a patient under the given hospital identity.
// Conflict target: (hospital, national_id) when national_id is present,
// else (hospital, passport_id).
func (r *Repository) Upsert(ctx context.Context, patient *domain.Patient) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if patient.Hospital == "" {
		return fmt.Errorf("upsert patient: %w: hospital required", apperr.ErrInvalidInput)
	}
	if (patient.NationalID == nil || *patient.NationalID == "") && (patient.PassportID == nil || *patient.PassportID == "") {
		return fmt.Errorf("upsert patient: %w: national_id or passport_id required", apperr.ErrInvalidInput)
	}

	query := upsertByNationalID
	if patient.NationalID == nil || *patient.NationalID == "" {
		query = upsertByPassportID
	}

	err := r.pool.QueryRow(ctx, query,
		patient.Hospital,
		patient.FirstNameTH, patient.MiddleNameTH, patient.LastNameTH,
		patient.FirstNameEN, patient.MiddleNameEN, patient.LastNameEN,
		patient.DateOfBirth, patient.PatientHN, patient.NationalID, patient.PassportID,
		patient.PhoneNumber, patient.Email, patient.Gender,
	).Scan(
		&patient.ID, &patient.Hospital,
		&patient.FirstNameTH, &patient.MiddleNameTH, &patient.LastNameTH,
		&patient.FirstNameEN, &patient.MiddleNameEN, &patient.LastNameEN,
		&patient.DateOfBirth, &patient.PatientHN, &patient.NationalID, &patient.PassportID,
		&patient.PhoneNumber, &patient.Email, &patient.Gender, &patient.CreatedAt, &patient.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert patient: %w", err)
	}
	return nil
}

const upsertInsert = `
		INSERT INTO patients (
			hospital,
			first_name_th, middle_name_th, last_name_th,
			first_name_en, middle_name_en, last_name_en,
			date_of_birth, patient_hn, national_id, passport_id,
			phone_number, email, gender
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14
		)
`

const upsertUpdate = `
		DO UPDATE SET
			first_name_th  = EXCLUDED.first_name_th,
			middle_name_th = EXCLUDED.middle_name_th,
			last_name_th   = EXCLUDED.last_name_th,
			first_name_en  = EXCLUDED.first_name_en,
			middle_name_en = EXCLUDED.middle_name_en,
			last_name_en   = EXCLUDED.last_name_en,
			date_of_birth  = EXCLUDED.date_of_birth,
			patient_hn     = EXCLUDED.patient_hn,
			national_id    = EXCLUDED.national_id,
			passport_id    = EXCLUDED.passport_id,
			phone_number   = EXCLUDED.phone_number,
			email          = EXCLUDED.email,
			gender         = EXCLUDED.gender,
			updated_at     = now()
		RETURNING ` + patientColumns

const upsertByNationalID = upsertInsert + `
		ON CONFLICT (hospital, national_id) WHERE national_id IS NOT NULL
` + upsertUpdate

const upsertByPassportID = upsertInsert + `
		ON CONFLICT (hospital, passport_id) WHERE national_id IS NULL AND passport_id IS NOT NULL
` + upsertUpdate

// Search returns patients matching criteria, always scoped to hospitalCode.
// hospitalCode is mandatory and never taken from criteria.
func (r *Repository) Search(ctx context.Context, hospitalCode string, criteria domain.SearchCriteria) ([]domain.Patient, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if hospitalCode == "" {
		return nil, fmt.Errorf("search patients: %w: hospital required", apperr.ErrInvalidInput)
	}

	limit := criteria.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := criteria.Offset
	if offset < 0 {
		offset = 0
	}

	var queryBuilder strings.Builder
	args := make([]any, 0, 12)
	paramIndex := 1

	queryBuilder.WriteString("SELECT ")
	queryBuilder.WriteString(patientColumns)
	queryBuilder.WriteString(" FROM patients WHERE hospital = $")
	queryBuilder.WriteString(strconv.Itoa(paramIndex))
	args = append(args, hospitalCode)
	paramIndex++

	addEquals := func(column string, value *string) {
		if value == nil || *value == "" {
			return
		}
		queryBuilder.WriteString(" AND ")
		queryBuilder.WriteString(column)
		queryBuilder.WriteString(" = $")
		queryBuilder.WriteString(strconv.Itoa(paramIndex))
		args = append(args, *value)
		paramIndex++
	}
	addCaseInsensitive := func(column string, value *string) {
		if value == nil || *value == "" {
			return
		}
		queryBuilder.WriteString(" AND lower(")
		queryBuilder.WriteString(column)
		queryBuilder.WriteString(") = lower($")
		queryBuilder.WriteString(strconv.Itoa(paramIndex))
		queryBuilder.WriteString(")")
		args = append(args, *value)
		paramIndex++
	}

	addEquals("national_id", criteria.NationalID)
	addEquals("passport_id", criteria.PassportID)
	addCaseInsensitive("first_name_en", criteria.FirstName)
	addCaseInsensitive("middle_name_en", criteria.MiddleName)
	addCaseInsensitive("last_name_en", criteria.LastName)
	if criteria.DateOfBirth != nil {
		queryBuilder.WriteString(" AND date_of_birth = $")
		queryBuilder.WriteString(strconv.Itoa(paramIndex))
		args = append(args, *criteria.DateOfBirth)
		paramIndex++
	}
	addEquals("phone_number", criteria.PhoneNumber)
	addCaseInsensitive("email", criteria.Email)

	queryBuilder.WriteString(" ORDER BY created_at DESC LIMIT $")
	queryBuilder.WriteString(strconv.Itoa(paramIndex))
	args = append(args, limit)
	paramIndex++
	queryBuilder.WriteString(" OFFSET $")
	queryBuilder.WriteString(strconv.Itoa(paramIndex))
	args = append(args, offset)

	rows, err := r.pool.Query(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	defer rows.Close()

	patients := make([]domain.Patient, 0)
	for rows.Next() {
		var patient domain.Patient
		if err := rows.Scan(
			&patient.ID, &patient.Hospital,
			&patient.FirstNameTH, &patient.MiddleNameTH, &patient.LastNameTH,
			&patient.FirstNameEN, &patient.MiddleNameEN, &patient.LastNameEN,
			&patient.DateOfBirth, &patient.PatientHN, &patient.NationalID, &patient.PassportID,
			&patient.PhoneNumber, &patient.Email, &patient.Gender, &patient.CreatedAt, &patient.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan patient: %w", err)
		}
		patients = append(patients, patient)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search patients rows: %w", err)
	}
	return patients, nil
}
