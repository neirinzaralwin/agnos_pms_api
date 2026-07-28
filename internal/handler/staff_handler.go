package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/neirinzaralwin/patient_management_system_api/internal/dto"
	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/service"
)

// StaffServicer is the staff service port used by handlers.
type StaffServicer interface {
	Create(ctx context.Context, username, password, hospital string) (*model.Staff, error)
	Login(ctx context.Context, username, password, hospital string) (*service.LoginResult, error)
}

// StaffHandler serves staff HTTP endpoints.
type StaffHandler struct {
	svc StaffServicer
}

// NewStaffHandler constructs staff HTTP handlers.
func NewStaffHandler(svc StaffServicer) *StaffHandler {
	return &StaffHandler{svc: svc}
}

// Create handles POST /staff/create.
func (h *StaffHandler) Create(c *gin.Context) {
	var req dto.CreateStaffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if isMaxBytesError(err) {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
			return
		}
		writeError(c, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	staff, err := h.svc.Create(c.Request.Context(), req.Username, req.Password, req.Hospital)
	if err != nil {
		mapError(c, err)
		return
	}

	writeData(c, http.StatusCreated, toStaffResponse(staff))
}

// Login handles POST /staff/login.
func (h *StaffHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		if isMaxBytesError(err) {
			writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body too large")
			return
		}
		writeError(c, http.StatusBadRequest, "INVALID_INPUT", "invalid request body")
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req.Username, req.Password, req.Hospital)
	if err != nil {
		mapError(c, err)
		return
	}

	writeData(c, http.StatusOK, dto.LoginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   result.ExpiresIn,
	})
}

func toStaffResponse(s *model.Staff) dto.StaffResponse {
	return dto.StaffResponse{
		ID:        s.ID,
		Username:  s.Username,
		Hospital:  s.Hospital,
		CreatedAt: s.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
