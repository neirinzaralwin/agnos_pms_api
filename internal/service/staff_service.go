package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

// StaffService is the application service for staff registration and authentication.
type StaffService struct {
	repo         StaffStore
	jwtSecret    string
	jwtTTL       time.Duration
	bcryptCost   int
	log          *slog.Logger
	now          func() time.Time
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

// Create registers a new staff member (application use case).
func (s *StaffService) Create(ctx context.Context, username, password, hospital string) (*model.Staff, error) {
	user, err := model.ParseUsername(username)
	if err != nil {
		return nil, err
	}
	hospitalCode, err := model.ParseHospitalCode(hospital)
	if err != nil {
		return nil, err
	}
	plain, err := model.ParsePassword(password)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain.String()), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	staff, err := model.NewStaff(user, hospitalCode, string(hash))
	if err != nil {
		return nil, err
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
	user, err := model.ParseUsername(username)
	if err != nil {
		// Keep login validation failures as unauthorized? Spec says missing field → 400.
		return nil, err
	}
	hospitalCode, err := model.ParseHospitalCode(hospital)
	if err != nil {
		return nil, err
	}

	if s.loginLimiter != nil {
		key := user.String() + "|" + hospitalCode.String()
		if !s.loginLimiter.Allow(key) {
			return nil, platform.ErrRateLimited
		}
	}

	staff, err := s.repo.GetByUsernameAndHospital(ctx, user.String(), hospitalCode.String())
	if err != nil {
		if errors.Is(err, platform.ErrNotFound) {
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

var dummyBcryptHash = mustDummyHash()

func mustDummyHash() string {
	h, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), 10)
	if err != nil {
		panic(err)
	}
	return string(h)
}
