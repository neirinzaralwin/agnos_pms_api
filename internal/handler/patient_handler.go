package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/neirinzaralwin/patient_management_system_api/internal/dto"
	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/service"
)

// PatientServicer is the patient service port used by handlers.
type PatientServicer interface {
	LookupFromHIS(ctx context.Context, hospital, id string) (*model.Patient, error)
	Search(ctx context.Context, hospital string, f service.SearchFilter) ([]model.Patient, error)
}

// PatientHandler serves patient HTTP endpoints.
type PatientHandler struct {
	svc PatientServicer
	log *slog.Logger
}

// NewPatientHandler constructs patient HTTP handlers.
func NewPatientHandler(svc PatientServicer, log *slog.Logger) *PatientHandler {
	if log == nil {
		log = slog.Default()
	}
	return &PatientHandler{svc: svc, log: log}
}

// Lookup handles GET /patient/search/:id (HIS lookup + upsert).
func (h *PatientHandler) Lookup(c *gin.Context) {
	c.Header("Cache-Control", "no-store")

	hospital := middleware.HospitalFromContext(c.Request.Context())
	staffID := middleware.StaffIDFromContext(c.Request.Context())
	id := c.Param("id")

	patient, err := h.svc.LookupFromHIS(c.Request.Context(), hospital, id)
	if err != nil {
		mapError(c, err)
		return
	}

	h.audit(c, staffID, hospital, c.FullPath(), 1, []string{"id"})
	writeData(c, http.StatusOK, toPatientResponse(patient))
}

// Search handles POST /patient/search (local DB only).
func (h *PatientHandler) Search(c *gin.Context) {
	c.Header("Cache-Control", "no-store")

	var req dto.PatientSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if isMaxBytesError(err) {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
			return
		}
		writeError(c, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	hospital := middleware.HospitalFromContext(c.Request.Context())
	staffID := middleware.StaffIDFromContext(c.Request.Context())

	filter := service.SearchFilter{
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
		mapError(c, err)
		return
	}

	out := make([]dto.PatientResponse, 0, len(patients))
	for i := range patients {
		out = append(out, toPatientResponse(&patients[i]))
	}

	h.audit(c, staffID, hospital, c.FullPath(), len(out), filter.FilterFieldNames())
	writeCollection(c, http.StatusOK, out, len(out))
}

func (h *PatientHandler) audit(c *gin.Context, staffID, hospital, endpoint string, count int, filters []string) {
	rid := middleware.RequestIDFromContext(c.Request.Context())
	h.log.Info("patient_access",
		"event", "patient_access",
		"staff_id", staffID,
		"hospital", hospital,
		"endpoint", endpoint,
		"result_count", count,
		"filters", filters,
		"request_id", rid,
		"ts", time.Now().UTC().Format(time.RFC3339),
	)
}

func toPatientResponse(p *model.Patient) dto.PatientResponse {
	var dob *string
	if p.DateOfBirth != nil {
		s := p.DateOfBirth.Format("2006-01-02")
		dob = &s
	}
	return dto.PatientResponse{
		ID:           p.ID,
		Hospital:     p.Hospital,
		FirstNameTH:  p.FirstNameTH,
		MiddleNameTH: p.MiddleNameTH,
		LastNameTH:   p.LastNameTH,
		FirstNameEN:  p.FirstNameEN,
		MiddleNameEN: p.MiddleNameEN,
		LastNameEN:   p.LastNameEN,
		DateOfBirth:  dob,
		PatientHN:    p.PatientHN,
		NationalID:   p.NationalID,
		PassportID:   p.PassportID,
		PhoneNumber:  p.PhoneNumber,
		Email:        p.Email,
		Gender:       p.Gender,
		CreatedAt:    p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:    p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
