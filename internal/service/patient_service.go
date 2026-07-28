package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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

// HISClient is the Hospital A port (anti-corruption layer entry).
type HISClient interface {
	SearchByID(ctx context.Context, id string) (*hospitala.Patient, error)
}

// PatientService is the application service for HIS lookup and hospital-scoped search.
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

// LookupFromHIS fetches a patient from HIS, registers it under the caller's hospital, and returns it.
func (s *PatientService) LookupFromHIS(ctx context.Context, hospital, id string) (*model.Patient, error) {
	hospitalCode, err := model.ParseHospitalCode(hospital)
	if err != nil {
		return nil, err
	}
	lookupID, err := model.ParseLookupID(id)
	if err != nil {
		return nil, err
	}

	hisPatient, err := s.his.SearchByID(ctx, lookupID.String())
	if err != nil {
		if errors.Is(err, platform.ErrNotFound) {
			return nil, platform.ErrNotFound
		}
		if errors.Is(err, platform.ErrInvalidInput) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", platform.ErrUpstream, err)
	}

	patient, err := model.RegisterFromHIS(hospitalCode, toHISPatientData(hisPatient))
	if err != nil {
		return nil, err
	}
	if err := s.repo.Upsert(ctx, patient); err != nil {
		return nil, fmt.Errorf("upsert patient: %w", err)
	}
	return patient, nil
}

// SearchFilter is kept as the handler-facing alias of model.SearchCriteriaInput.
type SearchFilter = model.SearchCriteriaInput

// Search queries local patients scoped to hospital. Never calls HIS.
func (s *PatientService) Search(ctx context.Context, hospital string, f SearchFilter) ([]model.Patient, error) {
	hospitalCode, err := model.ParseHospitalCode(hospital)
	if err != nil {
		return nil, err
	}
	criteria, err := model.ParseSearchCriteria(f)
	if err != nil {
		return nil, err
	}

	patients, err := s.repo.Search(ctx, hospitalCode.String(), repository.SearchFilter{
		NationalID:  criteria.NationalID,
		PassportID:  criteria.PassportID,
		FirstName:   criteria.FirstName,
		MiddleName:  criteria.MiddleName,
		LastName:    criteria.LastName,
		DateOfBirth: criteria.DateOfBirth,
		PhoneNumber: criteria.PhoneNumber,
		Email:       criteria.Email,
		Limit:       criteria.Limit,
		Offset:      criteria.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("search patients: %w", err)
	}
	if patients == nil {
		patients = []model.Patient{}
	}
	return patients, nil
}

func toHISPatientData(h *hospitala.Patient) model.HISPatientData {
	return model.HISPatientData{
		FirstNameTH:  h.FirstNameTH,
		MiddleNameTH: h.MiddleNameTH,
		LastNameTH:   h.LastNameTH,
		FirstNameEN:  h.FirstNameEN,
		MiddleNameEN: h.MiddleNameEN,
		LastNameEN:   h.LastNameEN,
		DateOfBirth:  h.DateOfBirth,
		PatientHN:    h.PatientHN,
		NationalID:   h.NationalID,
		PassportID:   h.PassportID,
		PhoneNumber:  h.PhoneNumber,
		Email:        h.Email,
		Gender:       h.Gender,
	}
}
