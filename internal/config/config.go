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
	// EnableDocs serves OpenAPI, Swagger UI, and ReDoc. Defaults to true
	// outside production; set ENABLE_DOCS=true|false to override.
	EnableDocs bool
}

// Load reads and validates required environment variables. It fails fast on
// missing or invalid secrets and URLs — never boots with secret defaults.
func Load() (Config, error) {
	appConfig := Config{
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

	if appConfig.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if _, operationError := url.Parse(appConfig.DatabaseURL); operationError != nil {
		return Config{}, fmt.Errorf("DATABASE_URL is invalid: %w", operationError)
	}

	if len(appConfig.JWTSecret) < minJWTSecretLen {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least %d characters", minJWTSecretLen)
	}

	ttl, operationError := parseDuration(getenv("JWT_TTL", "60m"), "JWT_TTL")
	if operationError != nil {
		return Config{}, operationError
	}
	appConfig.JWTTTL = ttl

	if rawValue := strings.TrimSpace(os.Getenv("BCRYPT_COST")); rawValue != "" {
		cost, operationError := strconv.Atoi(rawValue)
		if operationError != nil || cost < 10 || cost > 15 {
			return Config{}, fmt.Errorf("BCRYPT_COST must be an integer between 10 and 15")
		}
		appConfig.BcryptCost = cost
	}

	appConfig.HospitalABaseURL = strings.TrimSpace(os.Getenv("HOSPITAL_A_BASE_URL"))
	if appConfig.HospitalABaseURL == "" {
		return Config{}, fmt.Errorf("HOSPITAL_A_BASE_URL is required")
	}
	u, operationError := url.Parse(appConfig.HospitalABaseURL)
	if operationError != nil || u.Scheme == "" || u.Host == "" {
		return Config{}, fmt.Errorf("HOSPITAL_A_BASE_URL must be an absolute URL")
	}

	hisTimeout, operationError := parseDuration(getenv("HOSPITAL_A_TIMEOUT", "5s"), "HOSPITAL_A_TIMEOUT")
	if operationError != nil {
		return Config{}, operationError
	}
	appConfig.HospitalATimeout = hisTimeout

	if rawValue := strings.TrimSpace(os.Getenv("DB_MAX_CONNS")); rawValue != "" {
		n, operationError := strconv.Atoi(rawValue)
		if operationError != nil || n < 1 {
			return Config{}, fmt.Errorf("DB_MAX_CONNS must be a positive integer")
		}
		appConfig.DBMaxConns = int32(n)
	}
	if rawValue := strings.TrimSpace(os.Getenv("DB_MIN_CONNS")); rawValue != "" {
		n, operationError := strconv.Atoi(rawValue)
		if operationError != nil || n < 0 {
			return Config{}, fmt.Errorf("DB_MIN_CONNS must be a non-negative integer")
		}
		appConfig.DBMinConns = int32(n)
	}

	docs, operationError := parseEnableDocs(os.Getenv("ENABLE_DOCS"), appConfig.Env)
	if operationError != nil {
		return Config{}, operationError
	}
	appConfig.EnableDocs = docs

	return appConfig, nil
}

// parseEnableDocs returns whether developer docs routes should be mounted.
// ENABLE_DOCS=true|false overrides; otherwise docs are on unless ENV=production.
func parseEnableDocs(raw, env string) (bool, error) {
	rawValue := strings.TrimSpace(strings.ToLower(raw))
	switch rawValue {
	case "":
		return env != "production", nil
	case "1", "true", "yes", "on":
		return true, nil
	case "0", "false", "no", "off":
		return false, nil
	default:
		return false, fmt.Errorf("ENABLE_DOCS must be a boolean (true/false)")
	}
}

func getenv(key, fallback string) string {
	if rawValue := strings.TrimSpace(os.Getenv(key)); rawValue != "" {
		return rawValue
	}
	return fallback
}

func parseDuration(raw, name string) (time.Duration, error) {
	d, operationError := time.ParseDuration(raw)
	if operationError != nil || d <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration (e.g. 60m, 5s)", name)
	}
	return d, nil
}
