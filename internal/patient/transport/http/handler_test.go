package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/application"
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	patienttransport "github.com/neirinzaralwin/patient_management_system_api/internal/patient/transport/http"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type stubService struct {
	LookupFn func(ctx context.Context, hospital, lookupID string) (*domain.Patient, error)
	SearchFn func(ctx context.Context, hospital string, filter application.SearchFilter) ([]domain.Patient, error)
}

func (s *stubService) LookupFromHIS(ctx context.Context, hospital, lookupID string) (*domain.Patient, error) {
	return s.LookupFn(ctx, hospital, lookupID)
}
func (s *stubService) Search(ctx context.Context, hospital string, filter application.SearchFilter) ([]domain.Patient, error) {
	return s.SearchFn(ctx, hospital, filter)
}

func strPtr(s string) *string { return &s }

func authedRouter(secret string, h *patienttransport.Handler) *gin.Engine {
	r := gin.New()
	r.Use(middleware.RequestID())
	g := r.Group("/patient")
	g.Use(middleware.Auth(secret))
	g.GET("/search/:id", h.Lookup)
	g.POST("/search", h.Search)
	return r
}

func bearer(t *testing.T, secret, staffID, hospital string) string {
	t.Helper()
	tok, err := platform.Issue(platform.TokenClaims{StaffID: staffID, Hospital: hospital}, secret, time.Hour, time.Now())
	require.NoError(t, err)
	return tok
}

func TestLookup_OK(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	svc := &stubService{
		LookupFn: func(ctx context.Context, hospital, lookupID string) (*domain.Patient, error) {
			require.Equal(t, "hospital-a", hospital)
			return &domain.Patient{
				ID: "p1", Hospital: hospital, NationalID: strPtr(lookupID),
				FirstNameEN: strPtr("Somchai"),
				CreatedAt:   time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	r := authedRouter(secret, patienttransport.NewHandler(svc, nil))
	req := httptest.NewRequest(http.MethodGet, "/patient/search/1100700123456", nil)
	req.Header.Set("Authorization", "Bearer "+bearer(t, secret, "staff-1", "hospital-a"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	require.Contains(t, w.Body.String(), "Somchai")
}

func TestLookup_NoToken(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	r := authedRouter(secret, patienttransport.NewHandler(&stubService{}, nil))
	req := httptest.NewRequest(http.MethodGet, "/patient/search/1100700123456", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLookup_ExpiredToken(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	tok, err := platform.Issue(platform.TokenClaims{StaffID: "s1", Hospital: "hospital-a"}, secret, time.Minute, time.Now().Add(-2*time.Hour))
	require.NoError(t, err)
	r := authedRouter(secret, patienttransport.NewHandler(&stubService{}, nil))
	req := httptest.NewRequest(http.MethodGet, "/patient/search/1100700123456", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLookup_HIS404(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	svc := &stubService{
		LookupFn: func(ctx context.Context, hospital, lookupID string) (*domain.Patient, error) {
			return nil, platform.ErrNotFound
		},
	}
	r := authedRouter(secret, patienttransport.NewHandler(svc, nil))
	req := httptest.NewRequest(http.MethodGet, "/patient/search/MISSING", nil)
	req.Header.Set("Authorization", "Bearer "+bearer(t, secret, "s1", "hospital-a"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestLookup_HIS502(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	svc := &stubService{
		LookupFn: func(ctx context.Context, hospital, lookupID string) (*domain.Patient, error) {
			return nil, platform.ErrUpstream
		},
	}
	r := authedRouter(secret, patienttransport.NewHandler(svc, nil))
	req := httptest.NewRequest(http.MethodGet, "/patient/search/X", nil)
	req.Header.Set("Authorization", "Bearer "+bearer(t, secret, "s1", "hospital-a"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadGateway, w.Code)
}

func TestLookup_InvalidID(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	svc := &stubService{
		LookupFn: func(ctx context.Context, hospital, lookupID string) (*domain.Patient, error) {
			return nil, apperr.ErrInvalidInput
		},
	}
	r := authedRouter(secret, patienttransport.NewHandler(svc, nil))
	req := httptest.NewRequest(http.MethodGet, "/patient/search/bad!", nil)
	req.Header.Set("Authorization", "Bearer "+bearer(t, secret, "s1", "hospital-a"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearch_OK(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	svc := &stubService{
		SearchFn: func(ctx context.Context, hospital string, filter application.SearchFilter) ([]domain.Patient, error) {
			require.Equal(t, "hospital-a", hospital)
			return []domain.Patient{{
				ID: "p1", Hospital: hospital, Email: strPtr("a@example.com"),
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}}, nil
		},
	}
	r := authedRouter(secret, patienttransport.NewHandler(svc, nil))
	body := `{"email":"a@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/patient/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer(t, secret, "s1", "hospital-a"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	meta := resp["meta"].(map[string]any)
	require.Equal(t, float64(1), meta["count"])
}

func TestSearch_NoToken(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	r := authedRouter(secret, patienttransport.NewHandler(&stubService{}, nil))
	req := httptest.NewRequest(http.MethodPost, "/patient/search", bytes.NewBufferString(`{"email":"a@b.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestSearch_NoFilters(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	svc := &stubService{
		SearchFn: func(ctx context.Context, hospital string, filter application.SearchFilter) ([]domain.Patient, error) {
			return nil, apperr.ErrInvalidInput
		},
	}
	r := authedRouter(secret, patienttransport.NewHandler(svc, nil))
	req := httptest.NewRequest(http.MethodPost, "/patient/search", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer(t, secret, "s1", "hospital-a"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSearch_EmptyArrayNotNull(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	svc := &stubService{
		SearchFn: func(ctx context.Context, hospital string, filter application.SearchFilter) ([]domain.Patient, error) {
			return []domain.Patient{}, nil
		},
	}
	r := authedRouter(secret, patienttransport.NewHandler(svc, nil))
	req := httptest.NewRequest(http.MethodPost, "/patient/search", bytes.NewBufferString(`{"email":"none@x.com"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer(t, secret, "s1", "hospital-a"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"data":[]`)
	require.NotContains(t, w.Body.String(), `"data":null`)
}

func TestSearch_CrossHospitalIsolation_Handler(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	svc := &stubService{
		SearchFn: func(ctx context.Context, hospital string, filter application.SearchFilter) ([]domain.Patient, error) {
			require.Equal(t, "hospital-a", hospital)
			return []domain.Patient{}, nil
		},
	}
	r := authedRouter(secret, patienttransport.NewHandler(svc, nil))
	// Body tries to claim another hospital — ignored by design (not even in the DTO).
	body := `{"email":"a@example.com","hospital":"hospital-b"}`
	req := httptest.NewRequest(http.MethodPost, "/patient/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer(t, secret, "s1", "hospital-a"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"data":[]`)
}

func TestSearch_RepoError(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	svc := &stubService{
		SearchFn: func(ctx context.Context, hospital string, filter application.SearchFilter) ([]domain.Patient, error) {
			return nil, context.DeadlineExceeded
		},
	}
	r := authedRouter(secret, patienttransport.NewHandler(svc, nil))
	req := httptest.NewRequest(http.MethodPost, "/patient/search", bytes.NewBufferString(`{"email":"a@b.com"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bearer(t, secret, "s1", "hospital-a"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}
