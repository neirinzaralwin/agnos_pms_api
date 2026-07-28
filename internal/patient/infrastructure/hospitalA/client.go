// Package hospitalA is the anti-corruption layer for Hospital A's HIS. It is
// the only place that speaks Hospital A's wire format — SearchByID decodes
// the response and maps it into domain.HISPatientData before returning, so
// no other package ever imports Hospital A's JSON shape.
package hospitalA

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
	parsed, operationError := url.Parse(baseURL)
	if operationError != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("hospitalA: invalid base URL")
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
func (c *Client) SearchByID(requestContext context.Context, lookupID string) (*domain.HISPatientData, error) {
	if !ValidID(lookupID) {
		return nil, fmt.Errorf("%w: invalid id", apperr.ErrInvalidInput)
	}

	endpoint := c.baseURL + "/patient/search/" + url.PathEscape(lookupID)
	req, operationError := http.NewRequestWithContext(requestContext, http.MethodGet, endpoint, nil)
	if operationError != nil {
		return nil, fmt.Errorf("hospitalA: build request: %w", operationError)
	}

	resp, operationError := c.httpClient.Do(req)
	if operationError != nil {
		c.log.Warn("hospitalA request failed", "error", operationError.Error())
		return nil, fmt.Errorf("%w: %v", platform.ErrUpstream, operationError)
	}
	defer resp.Body.Close()

	body, operationError := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if operationError != nil {
		return nil, fmt.Errorf("%w: read body: %v", platform.ErrUpstream, operationError)
	}

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, platform.ErrNotFound
	case resp.StatusCode != http.StatusOK:
		c.log.Warn("hospitalA non-ok status", "status", resp.StatusCode, "id_masked", maskID(lookupID))
		return nil, fmt.Errorf("%w: status %d", platform.ErrUpstream, resp.StatusCode)
	}

	var payload wirePatient
	if operationError := json.Unmarshal(body, &payload); operationError != nil {
		c.log.Warn("hospitalA decode failure", "id_masked", maskID(lookupID))
		return nil, fmt.Errorf("%w: decode: %v", platform.ErrUpstream, operationError)
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
