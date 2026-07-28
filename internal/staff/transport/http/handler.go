// Package http adapts the staff application service to Gin. Handlers bind
// and respond; no SQL, no business rules.
package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/httpkit"
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/application"
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/domain"
)

// Service is the staff application port used by this handler.
type Service interface {
	Create(ctx context.Context, username, password, hospital string) (*domain.Staff, error)
	Login(ctx context.Context, username, password, hospital string) (*application.LoginResult, error)
}

// Handler serves the staff HTTP endpoints.
type Handler struct {
	svc Service
}

// NewHandler constructs staff HTTP handlers.
func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Create handles POST /staff/create.
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpkit.WriteBindError(c, err)
		return
	}

	staff, err := h.svc.Create(c.Request.Context(), req.Username, req.Password, req.Hospital)
	if err != nil {
		httpkit.MapError(c, err)
		return
	}

	httpkit.WriteData(c, http.StatusCreated, toStaffResponse(staff))
}

// Login handles POST /staff/login.
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpkit.WriteBindError(c, err)
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req.Username, req.Password, req.Hospital)
	if err != nil {
		httpkit.MapError(c, err)
		return
	}

	httpkit.WriteData(c, http.StatusOK, LoginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   result.ExpiresIn,
	})
}

func toStaffResponse(staff *domain.Staff) StaffResponse {
	return StaffResponse{
		ID:        staff.ID,
		Username:  staff.Username,
		Hospital:  staff.Hospital,
		CreatedAt: staff.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: staff.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
