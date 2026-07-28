package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/neirinzaralwin/patient_management_system_api/internal/client/hospitala"
	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/repository"
)

// PatientStore is the persistence port for patients.
type PatientStore interface {
	Upsert(ctx context.Context, p *model.Patient) error
	Search(ctx context.Context, hospital string, f repository.SearchFilter) ([]model.Patient, error)
}

// HISClient is the Hospital A port.
type HISClient interface {
	SearchByID(ctx context.Context, id string) (*hospitala.Patient, error)
}

// PatientService orchestrates HIS lookup and local search.
type PatientService struct {
	repo PatientStore
	his  HISClient
	log  *slog.Logger
}

// NewPatientService constructs a PatientService.
func NewPatientService(repo PatientStore, his HISClient, log *slog.Logger) *PatientService {
	if log == nil {
		log = slog.Default()
	}
	return &PatientService{repo: repo, his: his, log: log}
}

// LookupFromHIS fetches a patient from HIS, upserts under hospital, and returns it.
func (s *PatientService) LookupFromHIS(ctx context.Context, hospital, id string) (*model.Patient, error) {
	hospital = strings.ToLower(strings.TrimSpace(hospital))
	if hospital == "" {
		return nil, fmt.Errorf("%w: hospital required", platform.ErrInvalidInput)
	}
	if !hospitala.ValidID(id) {
		return nil, fmt.Errorf("%w: invalid id", platform.ErrInvalidInput)
	}

	hisPatient, err := s.his.SearchByID(ctx, id)
	if err != nil {
		if errors.Is(err, platform.ErrNotFound) {
			return nil, platform.ErrNotFound
		}
		if errors.Is(err, platform.ErrInvalidInput) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", platform.ErrUpstream, err)
	}

	patient := mapHISToModel(hospital, hisPatient)
	if err := s.repo.Upsert(ctx, patient); err != nil {
		return nil, fmt.Errorf("upsert patient: %w", err)
	}
	return patient, nil
}

// SearchFilter is the service-level search input (optional fields).
type SearchFilter struct {
	NationalID  *string
	PassportID  *string
	FirstName   *string
	MiddleName  *string
	LastName    *string
	DateOfBirth *string // YYYY-MM-DD
	PhoneNumber *string
	Email       *string
	Limit       *int
	Offset      *int
}

// FilterFieldNames returns which filter fields were set (names only, never values).
func (f SearchFilter) FilterFieldNames() []string {
	names := make([]string, 0, 8)
	add := func(name string, v *string) {
		if v != nil && *v != "" {
			names = append(names, name)
		}
	}
	add("national_id", f.NationalID)
	add("passport_id", f.PassportID)
	add("first_name", f.FirstName)
	add("middle_name", f.MiddleName)
	add("last_name", f.LastName)
	add("date_of_birth", f.DateOfBirth)
	add("phone_number", f.PhoneNumber)
	add("email", f.Email)
	return names
}

// Search queries local patients scoped to hospital. Never calls HIS.
func (s *PatientService) Search(ctx context.Context, hospital string, f SearchFilter) ([]model.Patient, error) {
	hospital = strings.ToLower(strings.TrimSpace(hospital))
	if hospital == "" {
		return nil, fmt.Errorf("%w: hospital required", platform.ErrInvalidInput)
	}
	if len(f.FilterFieldNames()) == 0 {
		return nil, fmt.Errorf("%w: at least one search filter is required", platform.ErrInvalidInput)
	}

	repoFilter := repository.SearchFilter{
		NationalID:  trimPtr(f.NationalID),
		PassportID:  trimPtr(f.PassportID),
		FirstName:   trimPtr(f.FirstName),
		MiddleName:  trimPtr(f.MiddleName),
		LastName:    trimPtr(f.LastName),
		PhoneNumber: trimPtr(f.PhoneNumber),
		Email:       trimPtr(f.Email),
	}
	if f.Limit != nil {
		repoFilter.Limit = *f.Limit
	}
	if f.Offset != nil {
		repoFilter.Offset = *f.Offset
	}
	if f.DateOfBirth != nil && *f.DateOfBirth != "" {
		d, err := time.Parse("2006-01-02", strings.TrimSpace(*f.DateOfBirth))
		if err != nil {
			return nil, fmt.Errorf("%w: date_of_birth must be YYYY-MM-DD", platform.ErrInvalidInput)
		}
		repoFilter.DateOfBirth = &d
	}

	patients, err := s.repo.Search(ctx, hospital, repoFilter)
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	if patients == nil {
		patients = []model.Patient{}
	}
	return patients, nil
}

func mapHISToModel(hospital string, h *hospitala.Patient) *model.Patient {
	p := &model.Patient{
		Hospital:     hospital,
		FirstNameTH:  emptyToNil(h.FirstNameTH),
		MiddleNameTH: emptyToNil(h.MiddleNameTH),
		LastNameTH:   emptyToNil(h.LastNameTH),
		FirstNameEN:  emptyToNil(h.FirstNameEN),
		MiddleNameEN: emptyToNil(h.MiddleNameEN),
		LastNameEN:   emptyToNil(h.LastNameEN),
		PatientHN:    emptyToNil(h.PatientHN),
		NationalID:   emptyToNil(h.NationalID),
		PassportID:   emptyToNil(h.PassportID),
		PhoneNumber:  emptyToNil(h.PhoneNumber),
		Email:        emptyToNil(h.Email),
		Gender:       normalizeGender(h.Gender),
	}
	if h.DateOfBirth != nil && *h.DateOfBirth != "" {
		if d, err := time.Parse("2006-01-02", strings.TrimSpace(*h.DateOfBirth)); err == nil {
			p.DateOfBirth = &d
		}
	}
	return p
}

func normalizeGender(g *string) *string {
	if g == nil {
		return nil
	}
	v := strings.TrimSpace(*g)
	if v == "M" || v == "F" {
		return &v
	}
	return nil
}

func emptyToNil(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}
