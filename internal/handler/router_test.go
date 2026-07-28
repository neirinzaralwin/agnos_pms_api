package handler_test

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/handler"
	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/service"
)

func TestNewRouter_WiresStaffAndPatient(t *testing.T) {
	secret := strings.Repeat("s", 32)
	staff := &stubStaffSvc{
		CreateFn: func(ctx context.Context, username, password, hospital string) (*model.Staff, error) {
			return &model.Staff{ID: "1", Username: username, Hospital: hospital, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
		LoginFn: func(ctx context.Context, username, password, hospital string) (*service.LoginResult, error) {
			tok, err := platform.Issue(platform.TokenClaims{StaffID: "1", Hospital: hospital}, secret, time.Hour, time.Now())
			require.NoError(t, err)
			return &service.LoginResult{AccessToken: tok, ExpiresIn: 3600}, nil
		},
	}
	patient := &stubPatientSvc{
		LookupFn: func(ctx context.Context, hospital, id string) (*model.Patient, error) {
			return &model.Patient{ID: "p1", Hospital: hospital, NationalID: strPtr(id), CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
		SearchFn: func(ctx context.Context, hospital string, f service.SearchFilter) ([]model.Patient, error) {
			return []model.Patient{}, nil
		},
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := handler.NewRouter(handler.RouterDeps{
		Log:            log,
		Pool:           nil,
		Env:            "test",
		JWTSecret:      secret,
		StaffHandler:   handler.NewStaffHandler(staff),
		PatientHandler: handler.NewPatientHandler(patient, log),
		LoginLimiter:   middleware.NewFixedWindowLimiter(100, time.Minute),
		PatientLimiter: middleware.NewFixedWindowLimiter(100, time.Minute),
	})

	// create
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(`{"username":"a","password":"password12345","hospital":"hospital-a"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// login
	req = httptest.NewRequest(http.MethodPost, "/staff/login", bytes.NewBufferString(`{"username":"a","password":"password12345","hospital":"hospital-a"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	tok := bearer(t, secret, "1", "hospital-a")
	req = httptest.NewRequest(http.MethodGet, "/patient/search/1100700123456", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/patient/search", bytes.NewBufferString(`{"email":"x@y.com"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
