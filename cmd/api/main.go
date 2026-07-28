package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/neirinzaralwin/patient_management_system_api/internal/config"
	"github.com/neirinzaralwin/patient_management_system_api/internal/httpapi"
	"github.com/neirinzaralwin/patient_management_system_api/internal/middleware"
	patientapp "github.com/neirinzaralwin/patient_management_system_api/internal/patient/application"
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/infrastructure/hospitalA"
	patientpostgres "github.com/neirinzaralwin/patient_management_system_api/internal/patient/infrastructure/postgres"
	patienttransport "github.com/neirinzaralwin/patient_management_system_api/internal/patient/transport/http"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	staffapp "github.com/neirinzaralwin/patient_management_system_api/internal/staff/application"
	staffpostgres "github.com/neirinzaralwin/patient_management_system_api/internal/staff/infrastructure/postgres"
	stafftransport "github.com/neirinzaralwin/patient_management_system_api/internal/staff/transport/http"
)

// version is injected via -ldflags at build time.
var version = "dev"

func main() {
	if operationError := run(); operationError != nil {
		fmt.Fprintf(os.Stderr, "startup error: %v\n", operationError)
		os.Exit(1)
	}
}

func run() error {
	// Load runtime configuration from the environment.
	appConfig, operationError := config.Load()
	if operationError != nil {
		return fmt.Errorf("config: %w", operationError)
	}

	// Initialize shared infrastructure.
	log := platform.NewLogger(appConfig.Env, appConfig.LogLevel)

	parentContext := context.Background()
	pool, operationError := platform.NewPool(parentContext, appConfig)
	if operationError != nil {
		return operationError
	}
	defer pool.Close()

	// Build external clients and persistence adapters.
	hisClient, operationError := hospitalA.New(appConfig.HospitalABaseURL, appConfig.HospitalATimeout, log)
	if operationError != nil {
		return fmt.Errorf("hospitalA client: %w", operationError)
	}

	staffRepo := staffpostgres.NewRepository(pool)
	patientRepo := patientpostgres.NewRepository(pool)

	// Wire application services and rate limits.
	staffSvc := staffapp.NewService(staffRepo, appConfig.JWTSecret, appConfig.JWTTTL, appConfig.BcryptCost, log)
	staffSvc.SetLoginLimiter(middleware.NewFixedWindowLimiter(10, 15*time.Minute))

	patientSvc := patientapp.NewService(patientRepo, hisClient, log)

	// Create HTTP handlers and compose the router.
	staffHandler := stafftransport.NewHandler(staffSvc)
	patientHandler := patienttransport.NewHandler(patientSvc, log)

	router := httpapi.NewRouter(httpapi.RouterDeps{
		Log:            log,
		Pool:           pool,
		Env:            appConfig.Env,
		JWTSecret:      appConfig.JWTSecret,
		EnableDocs:     appConfig.EnableDocs,
		StaffHandler:   staffHandler,
		PatientHandler: patientHandler,
		LoginLimiter:   middleware.NewFixedWindowLimiter(10, 15*time.Minute),
		PatientLimiter: middleware.NewFixedWindowLimiter(60, time.Minute),
	})

	// Configure the HTTP server with production-safe timeouts.
	addr := ":" + appConfig.Port
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Start serving requests in the background.
	serverErrorCh := make(chan error, 1)
	go func() {
		if operationError := srv.ListenAndServe(); operationError != nil && !errors.Is(operationError, http.ErrServerClosed) {
			serverErrorCh <- operationError
		}
	}()

	// Emit startup metadata for operators.
	log.Info("server started",
		"version", version,
		"port", appConfig.Port,
		"env", appConfig.Env,
		"log_level", appConfig.LogLevel,
		"docs", appConfig.EnableDocs,
	)

	// Wait for a shutdown signal or an unexpected server error.
	signalContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-signalContext.Done():
		log.Info("shutdown signal received")
	case operationError := <-serverErrorCh:
		return fmt.Errorf("server error: %w", operationError)
	}

	// Gracefully stop the server and allow inflight requests to finish.
	shutdownContext, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if operationError := srv.Shutdown(shutdownContext); operationError != nil {
		log.Error("graceful shutdown failed", "error", operationError)
		return operationError
	}
	log.Info("server stopped")
	return nil
}
