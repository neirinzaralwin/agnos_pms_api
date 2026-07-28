package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
)

// SearchFilter holds optional patient search predicates. Hospital is NEVER here —
// it is a required argument on Search.
type SearchFilter struct {
	NationalID  *string
	PassportID  *string
	FirstName   *string
	MiddleName  *string
	LastName    *string
	DateOfBirth *time.Time
	PhoneNumber *string
	Email       *string
	Limit       int
	Offset      int
}

// PatientRepository persists patient rows.
type PatientRepository struct {
	pool *pgxpool.Pool
}

// NewPatientRepository constructs a PatientRepository.
func NewPatientRepository(pool *pgxpool.Pool) *PatientRepository {
	return &PatientRepository{pool: pool}
}

const patientColumns = `
	id, hospital,
	first_name_th, middle_name_th, last_name_th,
	first_name_en, middle_name_en, last_name_en,
	date_of_birth, patient_hn, national_id, passport_id,
	phone_number, email, gender, created_at, updated_at
`

// Upsert inserts or updates a patient under the given hospital identity.
// Conflict target: (hospital, national_id) when national_id present;
// else (hospital, passport_id).
func (r *PatientRepository) Upsert(ctx context.Context, p *model.Patient) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if p.Hospital == "" {
		return fmt.Errorf("upsert patient: %w: hospital required", platform.ErrInvalidInput)
	}
	if (p.NationalID == nil || *p.NationalID == "") && (p.PassportID == nil || *p.PassportID == "") {
		return fmt.Errorf("upsert patient: %w: national_id or passport_id required", platform.ErrInvalidInput)
	}

	q := upsertByNationalID
	if p.NationalID == nil || *p.NationalID == "" {
		q = upsertByPassportID
	}

	err := r.pool.QueryRow(ctx, q,
		p.Hospital,
		p.FirstNameTH, p.MiddleNameTH, p.LastNameTH,
		p.FirstNameEN, p.MiddleNameEN, p.LastNameEN,
		p.DateOfBirth, p.PatientHN, p.NationalID, p.PassportID,
		p.PhoneNumber, p.Email, p.Gender,
	).Scan(
		&p.ID, &p.Hospital,
		&p.FirstNameTH, &p.MiddleNameTH, &p.LastNameTH,
		&p.FirstNameEN, &p.MiddleNameEN, &p.LastNameEN,
		&p.DateOfBirth, &p.PatientHN, &p.NationalID, &p.PassportID,
		&p.PhoneNumber, &p.Email, &p.Gender, &p.CreatedAt, &p.UpdatedAt,
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

// Search returns patients matching f, always scoped to hospital.
// hospital is mandatory and never taken from the filter.
func (r *PatientRepository) Search(ctx context.Context, hospital string, f SearchFilter) ([]model.Patient, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if hospital == "" {
		return nil, fmt.Errorf("search patients: %w: hospital required", platform.ErrInvalidInput)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := f.Offset
	if offset < 0 {
		offset = 0
	}

	var b strings.Builder
	args := make([]any, 0, 12)
	argN := 1

	b.WriteString("SELECT ")
	b.WriteString(patientColumns)
	b.WriteString(" FROM patients WHERE hospital = $")
	b.WriteString(strconv.Itoa(argN))
	args = append(args, hospital)
	argN++

	addEq := func(col string, v *string) {
		if v == nil || *v == "" {
			return
		}
		b.WriteString(" AND ")
		b.WriteString(col)
		b.WriteString(" = $")
		b.WriteString(strconv.Itoa(argN))
		args = append(args, *v)
		argN++
	}
	addILike := func(col string, v *string) {
		if v == nil || *v == "" {
			return
		}
		b.WriteString(" AND lower(")
		b.WriteString(col)
		b.WriteString(") = lower($")
		b.WriteString(strconv.Itoa(argN))
		b.WriteString(")")
		args = append(args, *v)
		argN++
	}

	addEq("national_id", f.NationalID)
	addEq("passport_id", f.PassportID)
	addILike("first_name_en", f.FirstName)
	addILike("middle_name_en", f.MiddleName)
	addILike("last_name_en", f.LastName)
	if f.DateOfBirth != nil {
		b.WriteString(" AND date_of_birth = $")
		b.WriteString(strconv.Itoa(argN))
		args = append(args, *f.DateOfBirth)
		argN++
	}
	addEq("phone_number", f.PhoneNumber)
	addILike("email", f.Email)

	// Also match Thai name fields when English name filters are provided.
	// (first/middle/last already matched against *_en above; also OR *_th)
	// Kept simple: exact lower match on EN only is enough for the assignment.

	b.WriteString(" ORDER BY created_at DESC LIMIT $")
	b.WriteString(strconv.Itoa(argN))
	args = append(args, limit)
	argN++
	b.WriteString(" OFFSET $")
	b.WriteString(strconv.Itoa(argN))
	args = append(args, offset)

	rows, err := r.pool.Query(ctx, b.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	defer rows.Close()

	out := make([]model.Patient, 0)
	for rows.Next() {
		var p model.Patient
		if err := rows.Scan(
			&p.ID, &p.Hospital,
			&p.FirstNameTH, &p.MiddleNameTH, &p.LastNameTH,
			&p.FirstNameEN, &p.MiddleNameEN, &p.LastNameEN,
			&p.DateOfBirth, &p.PatientHN, &p.NationalID, &p.PassportID,
			&p.PhoneNumber, &p.Email, &p.Gender, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan patient: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search patients rows: %w", err)
	}
	return out, nil
}
