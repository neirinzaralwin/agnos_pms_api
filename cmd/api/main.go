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
	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/infrastructure/hospitala"
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
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "startup error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	log := platform.NewLogger(cfg.Env, cfg.LogLevel)

	ctx := context.Background()
	pool, err := platform.NewPool(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	hisClient, err := hospitala.New(cfg.HospitalABaseURL, cfg.HospitalATimeout, log)
	if err != nil {
		return fmt.Errorf("hospitala client: %w", err)
	}

	staffRepo := staffpostgres.NewRepository(pool)
	patientRepo := patientpostgres.NewRepository(pool)

	staffSvc := staffapp.NewService(staffRepo, cfg.JWTSecret, cfg.JWTTTL, cfg.BcryptCost, log)
	staffSvc.SetLoginLimiter(middleware.NewFixedWindowLimiter(10, 15*time.Minute))

	patientSvc := patientapp.NewService(patientRepo, hisClient, log)

	staffHandler := stafftransport.NewHandler(staffSvc)
	patientHandler := patienttransport.NewHandler(patientSvc, log)

	router := httpapi.NewRouter(httpapi.RouterDeps{
		Log:            log,
		Pool:           pool,
		Env:            cfg.Env,
		JWTSecret:      cfg.JWTSecret,
		StaffHandler:   staffHandler,
		PatientHandler: patientHandler,
		LoginLimiter:   middleware.NewFixedWindowLimiter(10, 15*time.Minute),
		PatientLimiter: middleware.NewFixedWindowLimiter(60, time.Minute),
	})

	addr := ":" + cfg.Port
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	log.Info("server started",
		"version", version,
		"port", cfg.Port,
		"env", cfg.Env,
		"log_level", cfg.LogLevel,
	)

	sigCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-sigCtx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("server error: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
		return err
	}
	log.Info("server stopped")
	return nil
}
