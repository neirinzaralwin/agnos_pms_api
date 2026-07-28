package application

import (
	"context"
	"errors"
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
func (s *Service) LookupFromHIS(requestContext context.Context, hospitalCode, lookupID string) (*domain.Patient, error) {
	validHospital, operationError := hospital.Parse(hospitalCode)
	if operationError != nil {
		return nil, operationError
	}
	validLookupID, operationError := domain.ParseLookupID(lookupID)
	if operationError != nil {
		return nil, operationError
	}

	hisPatient, operationError := s.his.SearchByID(requestContext, validLookupID.String())
	if operationError != nil {
		if errors.Is(operationError, platform.ErrNotFound) {
			return nil, platform.ErrNotFound
		}
		if errors.Is(operationError, apperr.ErrInvalidInput) {
			return nil, operationError
		}
		if errors.Is(operationError, platform.ErrUpstream) {
			return nil, operationError
		}
		return nil, fmt.Errorf("%w: %v", platform.ErrUpstream, operationError)
	}

	patient, operationError := domain.RegisterFromHIS(validHospital, *hisPatient)
	if operationError != nil {
		return nil, operationError
	}
	if operationError := s.repo.Upsert(requestContext, patient); operationError != nil {
		return nil, fmt.Errorf("upsert patient: %w", operationError)
	}
	return patient, nil
}

// SearchFilter is the handler-facing alias of domain.SearchCriteriaInput.
type SearchFilter = domain.SearchCriteriaInput

// Search queries local patients scoped to hospitalCode. Never calls HIS —
// upstream latency and outages must not degrade this path.
func (s *Service) Search(requestContext context.Context, hospitalCode string, filter SearchFilter) ([]domain.Patient, error) {
	validHospital, operationError := hospital.Parse(hospitalCode)
	if operationError != nil {
		return nil, operationError
	}
	criteria, operationError := domain.ParseSearchCriteria(filter)
	if operationError != nil {
		return nil, operationError
	}

	patients, operationError := s.repo.Search(requestContext, validHospital.String(), criteria)
	if operationError != nil {
		return nil, fmt.Errorf("search patients: %w", operationError)
	}
	if patients == nil {
		patients = []domain.Patient{}
	}
	return patients, nil
}
