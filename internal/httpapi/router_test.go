package httpapi_test

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

	"github.com/neirinzaralwin/patient_management_system_api/internal/httpapi"
	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
	patientapp "github.com/neirinzaralwin/patient_management_system_api/internal/patient/application"
	patientdomain "github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	patienttransport "github.com/neirinzaralwin/patient_management_system_api/internal/patient/transport/http"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	staffapp "github.com/neirinzaralwin/patient_management_system_api/internal/staff/application"
	staffdomain "github.com/neirinzaralwin/patient_management_system_api/internal/staff/domain"
	stafftransport "github.com/neirinzaralwin/patient_management_system_api/internal/staff/transport/http"
)

// stubStaffService and stubPatientService are small self-contained fakes —
// router_test only checks that requests reach the right handler, so the
// richer service tests live in each context's own application package.
type stubStaffService struct {
	CreateFn func(requestContext context.Context, username, password, hospital string) (*staffdomain.Staff, error)
	LoginFn  func(requestContext context.Context, username, password, hospital string) (*staffapp.LoginResult, error)
}

func (s *stubStaffService) Create(requestContext context.Context, username, password, hospital string) (*staffdomain.Staff, error) {
	return s.CreateFn(requestContext, username, password, hospital)
}
func (s *stubStaffService) Login(requestContext context.Context, username, password, hospital string) (*staffapp.LoginResult, error) {
	return s.LoginFn(requestContext, username, password, hospital)
}

type stubPatientService struct {
	LookupFn func(requestContext context.Context, hospital, lookupID string) (*patientdomain.Patient, error)
	SearchFn func(requestContext context.Context, hospital string, filter patientapp.SearchFilter) ([]patientdomain.Patient, error)
}

func (s *stubPatientService) LookupFromHIS(requestContext context.Context, hospital, lookupID string) (*patientdomain.Patient, error) {
	return s.LookupFn(requestContext, hospital, lookupID)
}
func (s *stubPatientService) Search(requestContext context.Context, hospital string, filter patientapp.SearchFilter) ([]patientdomain.Patient, error) {
	return s.SearchFn(requestContext, hospital, filter)
}

func TestNewRouter_WiresStaffAndPatient(t *testing.T) {
	secret := strings.Repeat("s", 32)
	staffSvc := &stubStaffService{
		CreateFn: func(requestContext context.Context, username, password, hospital string) (*staffdomain.Staff, error) {
			return &staffdomain.Staff{ID: "1", Username: username, Hospital: hospital, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
		LoginFn: func(requestContext context.Context, username, password, hospital string) (*staffapp.LoginResult, error) {
			tok, operationError := platform.Issue(platform.TokenClaims{StaffID: "1", Hospital: hospital}, secret, time.Hour, time.Now())
			require.NoError(t, operationError)
			return &staffapp.LoginResult{AccessToken: tok, ExpiresIn: 3600}, nil
		},
	}
	patientSvc := &stubPatientService{
		LookupFn: func(requestContext context.Context, hospital, lookupID string) (*patientdomain.Patient, error) {
			id := lookupID
			return &patientdomain.Patient{ID: "p1", Hospital: hospital, NationalID: &id, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
		SearchFn: func(requestContext context.Context, hospital string, filter patientapp.SearchFilter) ([]patientdomain.Patient, error) {
			return []patientdomain.Patient{}, nil
		},
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := httpapi.NewRouter(httpapi.RouterDeps{
		Log:            log,
		Pool:           nil,
		Env:            "test",
		JWTSecret:      secret,
		StaffHandler:   stafftransport.NewHandler(staffSvc),
		PatientHandler: patienttransport.NewHandler(patientSvc, log),
		LoginLimiter:   middleware.NewFixedWindowLimiter(100, time.Minute),
		PatientLimiter: middleware.NewFixedWindowLimiter(100, time.Minute),
	})

	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(`{"username":"a","password":"password12345","hospital":"hospital-a"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/staff/login", bytes.NewBufferString(`{"username":"a","password":"password12345","hospital":"hospital-a"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	tok, operationError := platform.Issue(platform.TokenClaims{StaffID: "1", Hospital: "hospital-a"}, secret, time.Hour, time.Now())
	require.NoError(t, operationError)

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
