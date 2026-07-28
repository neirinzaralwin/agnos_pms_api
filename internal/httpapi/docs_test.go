package httpapi_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/docs/openapi"
	"github.com/neirinzaralwin/patient_management_system_api/internal/httpapi"
	patienttransport "github.com/neirinzaralwin/patient_management_system_api/internal/patient/transport/http"
	stafftransport "github.com/neirinzaralwin/patient_management_system_api/internal/staff/transport/http"
)

func newDocsTestRouter(t *testing.T, enableDocs bool) http.Handler {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return httpapi.NewRouter(httpapi.RouterDeps{
		Log:            log,
		Pool:           nil,
		Env:            "test",
		JWTSecret:      strings.Repeat("s", 32),
		EnableDocs:     enableDocs,
		StaffHandler:   stafftransport.NewHandler(&stubStaffService{}),
		PatientHandler: patienttransport.NewHandler(&stubPatientService{}, log),
	})
}

func TestDocsRoutes_Enabled(t *testing.T) {
	t.Parallel()
	r := newDocsTestRouter(t, true)

	t.Run("serves_openapi_spec", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/openapi/openapi.yaml", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Header().Get("Content-Type"), "application/yaml")
		require.Equal(t, string(openapi.Spec), w.Body.String())
		require.Contains(t, w.Body.String(), "openapi:")
		require.Contains(t, w.Body.String(), "/patient/search")
	})

	t.Run("serves_swagger_ui", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/docs/swagger", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Header().Get("Content-Type"), "text/html")
		require.Contains(t, w.Body.String(), "/openapi/openapi.yaml")
		require.Contains(t, w.Body.String(), "swagger-ui")
	})

	t.Run("serves_redoc", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/docs/redoc", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Header().Get("Content-Type"), "text/html")
		require.Contains(t, w.Body.String(), "/openapi/openapi.yaml")
		require.Contains(t, w.Body.String(), "redoc")
	})
}

func TestDocsRoutes_Disabled(t *testing.T) {
	t.Parallel()
	r := newDocsTestRouter(t, false)

	paths := []string{
		"/openapi/openapi.yaml",
		"/docs/swagger",
		"/docs/redoc",
	}
	for _, path := range paths {
		path := path
		t.Run("returns_404_for_"+strings.TrimPrefix(path, "/"), func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			require.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}
