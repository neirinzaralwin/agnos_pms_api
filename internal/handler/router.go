package handler

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
)

// NewRouter builds the Gin engine with global middleware and health routes.
func NewRouter(log *slog.Logger, pool *pgxpool.Pool, env string) *gin.Engine {
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	_ = r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})

	r.Use(
		middleware.Recovery(log),
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.Timeout(10*time.Second),
	)

	health := NewHealth(pool)
	r.GET("/healthz", health.Healthz)
	r.GET("/readyz", health.Readyz)

	return r
}
