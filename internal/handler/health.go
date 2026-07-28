package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Health handles liveness and readiness probes.
type Health struct {
	pool *pgxpool.Pool
}

// NewHealth constructs health handlers.
func NewHealth(pool *pgxpool.Pool) *Health {
	return &Health{pool: pool}
}

// Healthz is process liveness — no dependency checks.
func (h *Health) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz checks database reachability.
func (h *Health) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":     "unavailable",
			"dependency": "postgres",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
