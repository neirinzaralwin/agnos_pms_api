package middleware_test

import (
	"io"
	"log/slog"
	"time"

	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
)

func nilLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
}

func issueTestToken(secret string) (string, error) {
	return platform.Issue(platform.TokenClaims{
		StaffID:  "s1",
		Hospital: "hospital-a",
	}, secret, time.Hour, time.Now())
}
