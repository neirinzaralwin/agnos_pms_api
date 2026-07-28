// Package hospitala is the anti-corruption layer for Hospital A's HIS. It is
// the only place that speaks Hospital A's wire format — SearchByID decodes
// the response and maps it into domain.HISPatientData before returning, so
// no other package ever imports Hospital A's JSON shape.
package hospitala

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/neirinzaralwin/patient_management_system_api/internal/patient/domain"
	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
	"github.com/neirinzaralwin/patient_management_system_api/internal/shared/apperr"
)

const maxResponseBytes = 1 << 20 // 1 MiB

// Client talks to Hospital A's HIS and implements domain.HISClient.
type Client struct {
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger
}

// New constructs a Hospital A client. baseURL must be an absolute URL.
func New(baseURL string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("hospitala: invalid base URL")
	}
	if log == nil {
		log = slog.Default()
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: timeout},
		log:        log,
	}, nil
}

// SearchByID looks up a patient by national_id or passport_id. Returns
// platform.ErrNotFound on HIS 404, platform.ErrUpstream on any other failure.
func (c *Client) SearchByID(ctx context.Context, lookupID string) (*domain.HISPatientData, error) {
	if !ValidID(lookupID) {
		return nil, fmt.Errorf("%w: invalid id", apperr.ErrInvalidInput)
	}

	endpoint := c.baseURL + "/patient/search/" + url.PathEscape(lookupID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("hospitala: build request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Warn("hospitala request failed", "error", err.Error())
		return nil, fmt.Errorf("%w: %v", platform.ErrUpstream, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", platform.ErrUpstream, err)
	}

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, platform.ErrNotFound
	case resp.StatusCode != http.StatusOK:
		c.log.Warn("hospitala non-ok status", "status", resp.StatusCode, "id_masked", maskID(lookupID))
		return nil, fmt.Errorf("%w: status %d", platform.ErrUpstream, resp.StatusCode)
	}

	var payload wirePatient
	if err := json.Unmarshal(body, &payload); err != nil {
		c.log.Warn("hospitala decode failure", "id_masked", maskID(lookupID))
		return nil, fmt.Errorf("%w: decode: %v", platform.ErrUpstream, err)
	}
	return payload.toDomain(), nil
}

// ValidID reports whether id is alphanumeric and length-bounded (1-64).
func ValidID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, ch := range id {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) {
			return false
		}
	}
	return true
}

func maskID(id string) string {
	if len(id) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(id)-4) + id[len(id)-4:]
}
