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
)

func TestRecovery(t *testing.T) {
	t.Parallel()
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Recovery(nilLogger()))
	r.GET("/boom", func(c *gin.Context) { panic("explode") })

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
	require.NotContains(t, w.Body.String(), "explode")
}

func TestLogger_SkipsHealth(t *testing.T) {
	t.Parallel()
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(nilLogger()))
	r.GET("/healthz", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/ok", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
}

func TestTimeout(t *testing.T) {
	t.Parallel()
	r := gin.New()
	r.Use(middleware.Timeout(50 * time.Millisecond))
	r.GET("/x", func(c *gin.Context) {
		select {
		case <-c.Request.Context().Done():
			c.Status(http.StatusGatewayTimeout)
		case <-time.After(200 * time.Millisecond):
			c.Status(http.StatusOK)
		}
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusGatewayTimeout, w.Code)
}

func TestHandlePayloadTooLarge(t *testing.T) {
	t.Parallel()
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.HandlePayloadTooLarge())
	r.POST("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("hi"))
	req.ContentLength = (1 << 20) + 1
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
	require.Contains(t, w.Body.String(), "PAYLOAD_TOO_LARGE")
}

func TestBodyLimit_SetsNosniff(t *testing.T) {
	t.Parallel()
	r := gin.New()
	r.Use(middleware.BodyLimit())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
}

func TestRateLimitByStaff(t *testing.T) {
	t.Parallel()
	secret := strings.Repeat("s", 32)
	limiter := middleware.NewFixedWindowLimiter(1, time.Minute)
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Auth(secret), middleware.RateLimitByStaff(limiter))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	tok, err := issueTestToken(secret)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/x", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusTooManyRequests, w2.Code)
}
