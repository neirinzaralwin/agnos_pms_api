package config

import (
	"strings"
	"testing"
)

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("HOSPITAL_A_BASE_URL", "https://hospital-a.api.co.th")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

func TestLoad_ShortJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/pms")
	t.Setenv("JWT_SECRET", "too-short")
	t.Setenv("HOSPITAL_A_BASE_URL", "https://hospital-a.api.co.th")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for short JWT_SECRET")
	}
}

func TestLoad_Valid(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/pms")
	t.Setenv("JWT_SECRET", strings.Repeat("s", 32))
	t.Setenv("HOSPITAL_A_BASE_URL", "https://hospital-a.api.co.th")
	t.Setenv("JWT_TTL", "60m")
	t.Setenv("HOSPITAL_A_TIMEOUT", "5s")
	t.Setenv("ENV", "development")
	t.Setenv("PORT", "8080")
	t.Setenv("LOG_LEVEL", "info")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.BcryptCost != 12 {
		t.Fatalf("BcryptCost = %d, want 12", cfg.BcryptCost)
	}
}
