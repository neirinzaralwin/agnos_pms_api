// Package application orchestrates patient use cases: HIS lookup-and-upsert,
// and hospital-scoped local search. It depends on domain ports only — never
// on Gin, pgx, or the HIS transport package directly.
package application

import (
	"errors"
	"context"
	"fmt"
	"log/slog"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/hospital"
)

// Service is the application service for HIS lookup and hospital-scoped search.
type Service struct {
	repo domain.Repository
	his  domain.HISClient
	log  *slog.Logger
}

// NewService constructs a patient application Service.
func NewService(repo domain.Repository, his domain.HISClient, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{repo: repo, his: his, log: log}
}

// LookupFromHIS fetches a patient from HIS, registers it under the caller's
// hospital, upserts it, and returns the stored record. Never called from the
// local-search path — see Search below.
func (s *Service) LookupFromHIS(ctx context.Context, hospitalCode, lookupID string) (*domain.Patient, error) {
	validHospital, err := hospital.Parse(hospitalCode)
	if err != nil {
		return nil, err
	}
	validLookupID, err := domain.ParseLookupID(lookupID)
	if err != nil {
		return nil, err
	}

	hisPatient, err := s.his.SearchByID(ctx, validLookupID.String())
	if err != nil {
		if errors.Is(err, platform.ErrNotFound) {
			return nil, platform.ErrNotFound
		}
		if errors.Is(err, apperr.ErrInvalidInput) {
			return nil, err
		}
		if errors.Is(err, platform.ErrUpstream) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", platform.ErrUpstream, err)
	}

	patient, err := domain.RegisterFromHIS(validHospital, *hisPatient)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, patient); err != nil {
		return nil, fmt.Errorf("upsert patient: %w", err)
	}
	return patient, nil
}

// SearchFilter is the handler-facing alias of domain.SearchCriteriaInput.
type SearchFilter = domain.SearchCriteriaInput

// Search queries local patients scoped to hospitalCode. Never calls HIS —
// upstream latency and outages must not degrade this path.
func (s *Service) Search(ctx context.Context, hospitalCode string, filter SearchFilter) ([]domain.Patient, error) {
	validHospital, err := hospital.Parse(hospitalCode)
	if err != nil {
		return nil, err
	}
	criteria, err := domain.ParseSearchCriteria(filter)
	if err != nil {
		return nil, err
	}

	patients, err := s.repo.Search(ctx, validHospital.String(), criteria)
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	if patients == nil {
		patients = []domain.Patient{}
	}
	return patients, nil
}
