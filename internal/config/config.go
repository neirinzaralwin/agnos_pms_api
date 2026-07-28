package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const minJWTSecretLen = 32

// Config holds validated process configuration from the environment.
type Config struct {
	Env                 string
	Port                string
	LogLevel            string
	DatabaseURL         string
	JWTSecret           string
	JWTTTL              time.Duration
	BcryptCost          int
	HospitalABaseURL    string
	HospitalATimeout    time.Duration
	DBMaxConns          int32
	DBMinConns          int32
	DBMaxConnLifetime   time.Duration
	DBMaxConnIdleTime   time.Duration
	DBHealthCheckPeriod time.Duration
}

// Load reads and validates required environment variables. It fails fast on
// missing or invalid secrets and URLs — never boots with secret defaults.
func Load() (Config, error) {
	cfg := Config{
		Env:                 getenv("ENV", "development"),
		Port:                getenv("PORT", "8080"),
		LogLevel:            getenv("LOG_LEVEL", "info"),
		DatabaseURL:         strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:           os.Getenv("JWT_SECRET"),
		BcryptCost:          12,
		DBMaxConns:          10,
		DBMinConns:          2,
		DBMaxConnLifetime:   30 * time.Minute,
		DBMaxConnIdleTime:   5 * time.Minute,
		DBHealthCheckPeriod: time.Minute,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if _, err := url.Parse(cfg.DatabaseURL); err != nil {
		return Config{}, fmt.Errorf("DATABASE_URL is invalid: %w", err)
	}

	if len(cfg.JWTSecret) < minJWTSecretLen {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLen)
	}

	ttl, err := parseDuration(getenv("JWT_TTL", "60m"), "JWT_TTL")
	if err != nil {
		return Config{}, err
	}
	cfg.JWTTTL = ttl

	if v := strings.TrimSpace(os.Getenv("BCRYPT_COST")); v != "" {
		cost, err := strconv.Atoi(v)
		if err != nil || cost < 10 || cost > 15 {
			return Config{}, fmt.Errorf("BCRYPT_COST must be an integer between 10 and 15")
		}
		cfg.BcryptCost = cost
	}

	cfg.HospitalABaseURL = strings.TrimSpace(os.Getenv("HOSPITAL_A_BASE_URL"))
	if cfg.HospitalABaseURL == "" {
		return Config{}, fmt.Errorf("HOSPITAL_A_BASE_URL is required")
	}
	u, err := url.Parse(cfg.HospitalABaseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return Config{}, fmt.Errorf("HOSPITAL_A_BASE_URL must be an absolute URL")
	}

	hisTimeout, err := parseDuration(getenv("HOSPITAL_A_TIMEOUT", "5s"), "HOSPITAL_A_TIMEOUT")
	if err != nil {
		return Config{}, err
	}
	cfg.HospitalATimeout = hisTimeout

	if v := strings.TrimSpace(os.Getenv("DB_MAX_CONNS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return Config{}, fmt.Errorf("DB_MAX_CONNS must be a positive integer")
		}
		cfg.DBMaxConns = int32(n)
	}
	if v := strings.TrimSpace(os.Getenv("DB_MIN_CONNS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return Config{}, fmt.Errorf("DB_MIN_CONNS must be a non-negative integer")
		}
		cfg.DBMinConns = int32(n)
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func parseDuration(raw, name string) (time.Duration, error) {
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration (e.g. 60m, 5s)", name)
	}
	return d, nil
}
