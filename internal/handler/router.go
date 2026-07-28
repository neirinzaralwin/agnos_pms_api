package handler

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
)

// RouterDeps holds dependencies for route registration.
type RouterDeps struct {
	Log            *slog.Logger
	Pool           *pgxpool.Pool
	Env            string
	JWTSecret      string
	StaffHandler   *StaffHandler
	PatientHandler *PatientHandler
	LoginLimiter   *middleware.FixedWindowLimiter
	PatientLimiter *middleware.FixedWindowLimiter
}

// NewRouter builds the Gin engine with global middleware and all routes.
func NewRouter(deps RouterDeps) *gin.Engine {
	if deps.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	_ = r.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})

	r.Use(
		middleware.Recovery(deps.Log),
		middleware.RequestID(),
		middleware.Logger(deps.Log),
		middleware.Timeout(10*time.Second),
		middleware.BodyLimit(),
		middleware.HandlePayloadTooLarge(),
	)

	health := NewHealth(deps.Pool)
	r.GET("/healthz", health.Healthz)
	r.GET("/readyz", health.Readyz)

	staff := r.Group("/staff")
	{
		staff.POST("/create", deps.StaffHandler.Create)
		login := staff.Group("")
		if deps.LoginLimiter != nil {
			login.Use(middleware.RateLimitByIP(deps.LoginLimiter))
		}
		login.POST("/login", deps.StaffHandler.Login)
	}

	patient := r.Group("/patient")
	patient.Use(middleware.Auth(deps.JWTSecret))
	if deps.PatientLimiter != nil {
		patient.Use(middleware.RateLimitByStaff(deps.PatientLimiter))
	}
	{
		patient.GET("/search/:id", deps.PatientHandler.Lookup)
		patient.POST("/search", deps.PatientHandler.Search)
	}

	return r
}
