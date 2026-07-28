package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/neirinzaralwin/patient_management_system_api/internal/model"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
)

// StaffStore is the persistence port for staff.
type StaffStore interface {
	Create(ctx context.Context, s *model.Staff) error
	GetByUsernameAndHospital(ctx context.Context, username, hospital string) (*model.Staff, error)
}

// StaffService handles staff create and login.
type StaffService struct {
	repo       StaffStore
	jwtSecret  string
	jwtTTL     time.Duration
	bcryptCost int
	log        *slog.Logger
	now        func() time.Time
	// loginLimiter is optional; when set, keys are "username|hospital".
	loginLimiter LoginLimiter
}

// LoginLimiter bounds failed/attempted logins per (username, hospital).
type LoginLimiter interface {
	Allow(key string) bool
}

// NewStaffService constructs a StaffService.
func NewStaffService(
	repo StaffStore,
	jwtSecret string,
	jwtTTL time.Duration,
	bcryptCost int,
	log *slog.Logger,
) *StaffService {
	if log == nil {
		log = slog.Default()
	}
	return &StaffService{
		repo:       repo,
		jwtSecret:  jwtSecret,
		jwtTTL:     jwtTTL,
		bcryptCost: bcryptCost,
		log:        log,
		now:        time.Now,
	}
}

// SetLoginLimiter attaches a per-(username, hospital) rate limiter.
func (s *StaffService) SetLoginLimiter(l LoginLimiter) {
	s.loginLimiter = l
}

// Create registers a new staff member.
func (s *StaffService) Create(ctx context.Context, username, password, hospital string) (*model.Staff, error) {
	username = strings.TrimSpace(username)
	hospital = strings.ToLower(strings.TrimSpace(hospital))
	if username == "" || hospital == "" {
		return nil, fmt.Errorf("%w: username and hospital are required", platform.ErrInvalidInput)
	}
	if len(password) < 12 {
		return nil, fmt.Errorf("%w: password must be at least 12 characters", platform.ErrInvalidInput)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	staff := &model.Staff{
		Username:     username,
		PasswordHash: string(hash),
		Hospital:     hospital,
	}
	if err := s.repo.Create(ctx, staff); err != nil {
		if errors.Is(err, platform.ErrConflict) {
			return nil, platform.ErrConflict
		}
		return nil, fmt.Errorf("create staff: %w", err)
	}
	return staff, nil
}

// LoginResult is returned on successful authentication.
type LoginResult struct {
	AccessToken string
	ExpiresIn   int64
}

// Login authenticates a staff member. Unknown user, wrong hospital, and wrong
// password all return the same platform.ErrUnauthorized (anti-enumeration).
func (s *StaffService) Login(ctx context.Context, username, password, hospital string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	hospital = strings.ToLower(strings.TrimSpace(hospital))

	if s.loginLimiter != nil {
		key := username + "|" + hospital
		if !s.loginLimiter.Allow(key) {
			return nil, platform.ErrRateLimited
		}
	}

	staff, err := s.repo.GetByUsernameAndHospital(ctx, username, hospital)
	if err != nil {
		if errors.Is(err, platform.ErrNotFound) {
			// Dummy compare so timing does not reveal existence.
			_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
			return nil, platform.ErrUnauthorized
		}
		return nil, fmt.Errorf("login: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(staff.PasswordHash), []byte(password)); err != nil {
		return nil, platform.ErrUnauthorized
	}

	now := s.now()
	token, err := platform.Issue(platform.TokenClaims{
		StaffID:  staff.ID,
		Hospital: staff.Hospital,
	}, s.jwtSecret, s.jwtTTL, now)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &LoginResult{
		AccessToken: token,
		ExpiresIn:   int64(s.jwtTTL.Seconds()),
	}, nil
}

// Precomputed bcrypt of "dummy-password-for-timing" at cost 10 — used only for
// constant-time padding when the staff row is missing.
var dummyBcryptHash = mustDummyHash()

func mustDummyHash() string {
	h, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), 10)
	if err != nil {
		panic(err)
	}
	return string(h)
}
