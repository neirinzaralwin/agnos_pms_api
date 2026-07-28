// Package httpapi is the composition root for HTTP transport: it wires
// global middleware and routes each bounded context's handler onto its
// endpoints. It does not contain business logic.
package httpapi

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
	patienttransport "github.com/neirinzaralwin/patient_management_system_api/internal/patient/transport/http"
	stafftransport "github.com/neirinzaralwin/patient_management_system_api/internal/staff/transport/http"
)

// RouterDeps holds dependencies for route registration.
type RouterDeps struct {
	Log            *slog.Logger
	Pool           *pgxpool.Pool
	Env            string
	JWTSecret      string
	EnableDocs     bool
	StaffHandler   *stafftransport.Handler
	PatientHandler *patienttransport.Handler
	LoginLimiter   *middleware.FixedWindowLimiter
	PatientLimiter *middleware.FixedWindowLimiter
}

// NewRouter builds the Gin engine with global middleware and all routes.
func NewRouter(deps RouterDeps) *gin.Engine {
	if deps.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	_ = engine.SetTrustedProxies([]string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})

	engine.Use(
		middleware.Recovery(deps.Log),
		middleware.RequestID(),
		middleware.Logger(deps.Log),
		middleware.Timeout(10*time.Second),
		middleware.BodyLimit(),
		middleware.HandlePayloadTooLarge(),
	)

	health := NewHealth(deps.Pool)
	engine.GET("/healthz", health.Healthz)
	engine.GET("/readyz", health.Readyz)

	if deps.EnableDocs {
		registerDocs(engine)
	}

	staff := engine.Group("/staff")
	{
		staff.POST("/create", deps.StaffHandler.Create)
		login := staff.Group("")
		if deps.LoginLimiter != nil {
			login.Use(middleware.RateLimitByIP(deps.LoginLimiter))
		}
		login.POST("/login", deps.StaffHandler.Login)
	}

	patient := engine.Group("/patient")
	patient.Use(middleware.Auth(deps.JWTSecret))
	if deps.PatientLimiter != nil {
		patient.Use(middleware.RateLimitByStaff(deps.PatientLimiter))
	}
	{
		patient.GET("/search/:id", deps.PatientHandler.Lookup)
		patient.POST("/search", deps.PatientHandler.Search)
	}

	return engine
}
