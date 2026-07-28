// Package application orchestrates staff use cases: registration and
// authentication. It depends on the domain and on ports (Repository,
// LoginLimiter) — never on Gin or a concrete database driver.
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/hospital"
	"github.com/neirinzaralwin/patient_management_system_api/internal/staff/domain"
)

// LoginLimiter bounds login attempts per (username, hospital) key.
type LoginLimiter interface {
	Allow(key string) bool
}

// Service is the application service for staff registration and login.
type Service struct {
	repo         domain.Repository
	jwtSecret    string
	jwtTTL       time.Duration
	bcryptCost   int
	log          *slog.Logger
	now          func() time.Time
	loginLimiter LoginLimiter
}

// NewService constructs a staff application Service.
func NewService(repo domain.Repository, jwtSecret string, jwtTTL time.Duration, bcryptCost int, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		repo:       repo,
		jwtSecret:  jwtSecret,
		jwtTTL:     jwtTTL,
		bcryptCost: bcryptCost,
		log:        log,
		now:        time.Now,
	}
}

// SetLoginLimiter attaches a per-(username, hospital) rate limiter.
func (s *Service) SetLoginLimiter(limiter LoginLimiter) {
	s.loginLimiter = limiter
}

// Create registers a new staff member.
func (s *Service) Create(requestContext context.Context, username, password, hospitalCode string) (*domain.Staff, error) {
	validUsername, operationError := domain.ParseUsername(username)
	if operationError != nil {
		return nil, operationError
	}
	validHospital, operationError := hospital.Parse(hospitalCode)
	if operationError != nil {
		return nil, operationError
	}
	validPassword, operationError := domain.ParsePassword(password)
	if operationError != nil {
		return nil, operationError
	}

	hash, operationError := bcrypt.GenerateFromPassword([]byte(validPassword.String()), s.bcryptCost)
	if operationError != nil {
		return nil, fmt.Errorf("hash password: %w", operationError)
	}

	staff, operationError := domain.New(validUsername, validHospital, string(hash))
	if operationError != nil {
		return nil, operationError
	}
	if operationError := s.repo.Create(requestContext, staff); operationError != nil {
		if errors.Is(operationError, platform.ErrConflict) {
			return nil, platform.ErrConflict
		}
		return nil, fmt.Errorf("create staff: %w", operationError)
	}
	return staff, nil
}

// LoginResult is returned on successful authentication.
type LoginResult struct {
	AccessToken string
	ExpiresIn   int64
}

// dummyHash absorbs the bcrypt comparison cost on an unknown-user login so
// that unknown-user and wrong-password failures take the same time
// (anti-enumeration).
var dummyHash = mustHash("dummy-password-for-timing")

func mustHash(password string) string {
	hash, operationError := bcrypt.GenerateFromPassword([]byte(password), 10)
	if operationError != nil {
		panic(operationError)
	}
	return string(hash)
}

// Login authenticates a staff member. Unknown user, wrong hospital, and
// wrong password all return the same platform.ErrUnauthorized.
func (s *Service) Login(requestContext context.Context, username, password, hospitalCode string) (*LoginResult, error) {
	validUsername, operationError := domain.ParseUsername(username)
	if operationError != nil {
		return nil, operationError
	}
	validHospital, operationError := hospital.Parse(hospitalCode)
	if operationError != nil {
		return nil, operationError
	}

	if s.loginLimiter != nil {
		key := validUsername.String() + "|" + validHospital.String()
		if !s.loginLimiter.Allow(key) {
			return nil, platform.ErrRateLimited
		}
	}

	staff, operationError := s.repo.FindByUsernameAndHospital(requestContext, validUsername.String(), validHospital.String())
	if operationError != nil {
		if errors.Is(operationError, platform.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
			return nil, platform.ErrUnauthorized
		}
		return nil, fmt.Errorf("login: %w", operationError)
	}

	if operationError := bcrypt.CompareHashAndPassword([]byte(staff.PasswordHash), []byte(password)); operationError != nil {
		return nil, platform.ErrUnauthorized
	}

	issuedAt := s.now()
	token, operationError := platform.Issue(platform.TokenClaims{
		StaffID:  staff.ID,
		Hospital: staff.Hospital,
	}, s.jwtSecret, s.jwtTTL, issuedAt)
	if operationError != nil {
		return nil, fmt.Errorf("issue token: %w", operationError)
	}

	return &LoginResult{
		AccessToken: token,
		ExpiresIn:   int64(s.jwtTTL.Seconds()),
	}, nil
}
