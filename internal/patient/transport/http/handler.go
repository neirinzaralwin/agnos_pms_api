// Package http adapts the patient application service to Gin. Handlers bind,
// call the service, and respond — no SQL, no HIS calls.
package http

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/application"
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/httpkit"
)

// Service is the patient application port used by this handler.
type Service interface {
	LookupFromHIS(ctx context.Context, hospital, lookupID string) (*domain.Patient, error)
	Search(ctx context.Context, hospital string, filter application.SearchFilter) ([]domain.Patient, error)
}

// Handler serves the patient HTTP endpoints.
type Handler struct {
	svc Service
	log *slog.Logger
}

// NewHandler constructs patient HTTP handlers.
func NewHandler(svc Service, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}
	return &Handler{svc: svc, log: log}
}

// Lookup handles GET /patient/search/:id (HIS lookup + upsert).
func (h *Handler) Lookup(c *gin.Context) {
	c.Header("Cache-Control", "no-store")

	hospital := middleware.HospitalFromContext(c.Request.Context())
	staffID := middleware.StaffIDFromContext(c.Request.Context())
	lookupID := c.Param("id")

	patient, err := h.svc.LookupFromHIS(c.Request.Context(), hospital, lookupID)
	if err != nil {
		httpkit.MapError(c, err)
		return
	}

	h.audit(c, staffID, hospital, c.FullPath(), 1, []string{"id"})
	httpkit.WriteData(c, http.StatusOK, toResponse(patient))
}

// Search handles POST /patient/search (local DB only).
func (h *Handler) Search(c *gin.Context) {
	c.Header("Cache-Control", "no-store")

	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpkit.WriteBindError(c, err)
		return
	}

	hospital := middleware.HospitalFromContext(c.Request.Context())
	staffID := middleware.StaffIDFromContext(c.Request.Context())

	filter := application.SearchFilter{
		NationalID:  req.NationalID,
		PassportID:  req.PassportID,
		FirstName:   req.FirstName,
		MiddleName:  req.MiddleName,
		LastName:    req.LastName,
		DateOfBirth: req.DateOfBirth,
		PhoneNumber: req.PhoneNumber,
		Email:       req.Email,
		Limit:       req.Limit,
		Offset:      req.Offset,
	}

	patients, err := h.svc.Search(c.Request.Context(), hospital, filter)
	if err != nil {
		httpkit.MapError(c, err)
		return
	}

	out := make([]Response, 0, len(patients))
	for i := range patients {
		out = append(out, toResponse(&patients[i]))
	}

	h.audit(c, staffID, hospital, c.FullPath(), len(out), filter.FilterFieldNames())
	httpkit.WriteCollection(c, http.StatusOK, out, len(out))
}

func (h *Handler) audit(c *gin.Context, staffID, hospital, endpoint string, resultCount int, filterFields []string) {
	requestID := middleware.RequestIDFromContext(c.Request.Context())
	h.log.Info("patient_access",
		"event", "patient_access",
		"staff_id", staffID,
		"hospital", hospital,
		"endpoint", endpoint,
		"result_count", resultCount,
		"filters", filterFields,
		"request_id", requestID,
		"ts", time.Now().UTC().Format(time.RFC3339),
	)
}

func toResponse(patient *domain.Patient) Response {
	var dob *string
	if patient.DateOfBirth != nil {
		formatted := patient.DateOfBirth.Format("2006-01-02")
		dob = &formatted
	}
	return Response{
		ID:           patient.ID,
		Hospital:     patient.Hospital,
		FirstNameTH:  patient.FirstNameTH,
		MiddleNameTH: patient.MiddleNameTH,
		LastNameTH:   patient.LastNameTH,
		FirstNameEN:  patient.FirstNameEN,
		MiddleNameEN: patient.MiddleNameEN,
		LastNameEN:   patient.LastNameEN,
		DateOfBirth:  dob,
		PatientHN:    patient.PatientHN,
		NationalID:   patient.NationalID,
		PassportID:   patient.PassportID,
		PhoneNumber:  patient.PhoneNumber,
		Email:        patient.Email,
		Gender:       patient.Gender,
		CreatedAt:    patient.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    patient.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
