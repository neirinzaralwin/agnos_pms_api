package handler_test

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

	"github.com/neirinzaralwin/patient_management_system_api/internal/handler"
	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type stubStaffSvc struct {
	CreateFn func(ctx context.Context, username, password, hospital string) (*model.Staff, error)
	LoginFn  func(ctx context.Context, username, password, hospital string) (*service.LoginResult, error)
}

func (s *stubStaffSvc) Create(ctx context.Context, username, password, hospital string) (*model.Staff, error) {
	return s.CreateFn(ctx, username, password, hospital)
}
func (s *stubStaffSvc) Login(ctx context.Context, username, password, hospital string) (*service.LoginResult, error) {
	return s.LoginFn(ctx, username, password, hospital)
}

func TestStaffCreate_Handler(t *testing.T) {
	t.Parallel()
	svc := &stubStaffSvc{
		CreateFn: func(ctx context.Context, username, password, hospital string) (*model.Staff, error) {
			return &model.Staff{
				ID: "s1", Username: username, Hospital: hospital,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}, nil
		},
	}
	h := handler.NewStaffHandler(svc)
	r := gin.New()
	r.POST("/staff/create", h.Create)

	body := `{"username":"alice","password":"password12345","hospital":"hospital-a"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotContains(t, w.Body.String(), "password")
	require.Contains(t, w.Body.String(), `"username":"alice"`)
}

func TestStaffCreate_Conflict(t *testing.T) {
	t.Parallel()
	svc := &stubStaffSvc{
		CreateFn: func(ctx context.Context, username, password, hospital string) (*model.Staff, error) {
			return nil, platform.ErrConflict
		},
	}
	h := handler.NewStaffHandler(svc)
	r := gin.New()
	r.POST("/staff/create", h.Create)

	body := `{"username":"alice","password":"password12345","hospital":"hospital-a"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusConflict, w.Code)
	require.Contains(t, w.Body.String(), "CONFLICT")
}

func TestStaffCreate_MissingField(t *testing.T) {
	t.Parallel()
	h := handler.NewStaffHandler(&stubStaffSvc{})
	r := gin.New()
	r.POST("/staff/create", h.Create)

	body := `{"username":"alice","hospital":"hospital-a"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "INVALID_INPUT")
}

func TestStaffCreate_MalformedJSON(t *testing.T) {
	t.Parallel()
	h := handler.NewStaffHandler(&stubStaffSvc{})
	r := gin.New()
	r.POST("/staff/create", h.Create)

	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(`{`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStaffCreate_ShortPassword(t *testing.T) {
	t.Parallel()
	h := handler.NewStaffHandler(&stubStaffSvc{})
	r := gin.New()
	r.POST("/staff/create", h.Create)

	body := `{"username":"alice","password":"short","hospital":"hospital-a"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestStaffCreate_RepoError(t *testing.T) {
	t.Parallel()
	svc := &stubStaffSvc{
		CreateFn: func(ctx context.Context, username, password, hospital string) (*model.Staff, error) {
			return nil, context.DeadlineExceeded
		},
	}
	h := handler.NewStaffHandler(svc)
	r := gin.New()
	r.POST("/staff/create", h.Create)

	body := `{"username":"alice","password":"password12345","hospital":"hospital-a"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/create", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
	require.NotContains(t, w.Body.String(), "Deadline")
}

func TestStaffLogin_Handler(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	token, err := platform.Issue(platform.TokenClaims{StaffID: "s1", Hospital: "hospital-a"}, secret, time.Hour, time.Now())
	require.NoError(t, err)

	svc := &stubStaffSvc{
		LoginFn: func(ctx context.Context, username, password, hospital string) (*service.LoginResult, error) {
			return &service.LoginResult{AccessToken: token, ExpiresIn: 3600}, nil
		},
	}
	h := handler.NewStaffHandler(svc)
	r := gin.New()
	r.POST("/staff/login", h.Login)

	body := `{"username":"alice","password":"password12345","hospital":"hospital-a"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	require.Equal(t, "Bearer", data["token_type"])
	require.Equal(t, float64(3600), data["expires_in"])
}

func TestStaffLogin_UnauthorizedIdenticalBodies(t *testing.T) {
	t.Parallel()
	svc := &stubStaffSvc{
		LoginFn: func(ctx context.Context, username, password, hospital string) (*service.LoginResult, error) {
			return nil, platform.ErrUnauthorized
		},
	}
	h := handler.NewStaffHandler(svc)
	r := gin.New()
	r.POST("/staff/login", h.Login)

	bodies := []string{
		`{"username":"alice","password":"wrong-password","hospital":"hospital-a"}`,
		`{"username":"nobody","password":"password12345","hospital":"hospital-a"}`,
		`{"username":"alice","password":"password12345","hospital":"hospital-b"}`,
	}
	var first string
	for _, body := range bodies {
		req := httptest.NewRequest(http.MethodPost, "/staff/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
		if first == "" {
			first = w.Body.String()
		} else {
			require.Equal(t, first, w.Body.String())
		}
	}
}

func TestStaffLogin_MissingField(t *testing.T) {
	t.Parallel()
	h := handler.NewStaffHandler(&stubStaffSvc{})
	r := gin.New()
	r.POST("/staff/login", h.Login)

	body := `{"username":"alice","hospital":"hospital-a"}`
	req := httptest.NewRequest(http.MethodPost, "/staff/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
