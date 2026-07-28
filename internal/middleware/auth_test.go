package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuth_ValidToken(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	tok, operationError := platform.Issue(platform.TokenClaims{StaffID: "s1", Hospital: "hospital-a"}, secret, time.Hour, time.Now())
	require.NoError(t, operationError)

	r := gin.New()
	r.Use(middleware.Auth(secret))
	r.GET("/x", func(c *gin.Context) {
		require.Equal(t, "s1", middleware.StaffIDFromContext(c.Request.Context()))
		require.Equal(t, "hospital-a", middleware.HospitalFromContext(c.Request.Context()))
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestAuth_MissingToken(t *testing.T) {
	t.Parallel()
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Auth(strings.Repeat("s", 32)))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_MalformedToken(t *testing.T) {
	t.Parallel()
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Auth(strings.Repeat("s", 32)))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRateLimitByIP(t *testing.T) {
	t.Parallel()
	limiter := middleware.NewFixedWindowLimiter(2, time.Minute)
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.RateLimitByIP(limiter))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusTooManyRequests, w.Code)
	require.Contains(t, w.Body.String(), "RATE_LIMITED")
}
