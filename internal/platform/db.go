package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/neirinzaralwin/patient_management_system_api/internal/config"
)

// NewPool opens a pgxpool configured from appConfig and verifies connectivity with Ping.
func NewPool(requestContext context.Context, appConfig config.Config) (*pgxpool.Pool, error) {
	poolConfig, operationError := pgxpool.ParseConfig(appConfig.DatabaseURL)
	if operationError != nil {
		return nil, fmt.Errorf("parse database url: %w", operationError)
	}

	poolConfig.MaxConns = appConfig.DBMaxConns
	poolConfig.MinConns = appConfig.DBMinConns
	poolConfig.MaxConnLifetime = appConfig.DBMaxConnLifetime
	poolConfig.MaxConnIdleTime = appConfig.DBMaxConnIdleTime
	poolConfig.HealthCheckPeriod = appConfig.DBHealthCheckPeriod

	pool, operationError := pgxpool.NewWithConfig(requestContext, poolConfig)
	if operationError != nil {
		return nil, fmt.Errorf("open database pool: %w", operationError)
	}

	pingContext, cancel := context.WithTimeout(requestContext, 5*time.Second)
	defer cancel()
	if operationError := pool.Ping(pingContext); operationError != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", operationError)
	}

	return pool, nil
}
