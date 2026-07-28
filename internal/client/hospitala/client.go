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

	"github.com/neirinzaralwin/patient_management_system_api/internal/platform"
)

const maxResponseBytes = 1 << 20 // 1 MiB

// Client talks to Hospital A HIS.
type Client struct {
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger
}

// New constructs a Hospital A client. baseURL must be an absolute URL (no trailing path required).
func New(baseURL string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("hospitala: invalid base URL")
	}
	if log == nil {
		log = slog.Default()
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log,
	}, nil
}

// SearchByID looks up a patient by national_id or passport_id.
// Returns platform.ErrNotFound on HIS 404, platform.ErrUpstream on other failures.
func (c *Client) SearchByID(ctx context.Context, id string) (*Patient, error) {
	if !ValidID(id) {
		return nil, fmt.Errorf("%w: invalid id", platform.ErrInvalidInput)
	}

	endpoint := c.baseURL + "/patient/search/" + url.PathEscape(id)
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

	limited := io.LimitReader(resp.Body, maxResponseBytes)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("%w: read body: %v", platform.ErrUpstream, err)
	}

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, platform.ErrNotFound
	case resp.StatusCode != http.StatusOK:
		c.log.Warn("hospitala non-ok status",
			"status", resp.StatusCode,
			"id_masked", maskID(id),
		)
		return nil, fmt.Errorf("%w: status %d", platform.ErrUpstream, resp.StatusCode)
	}

	var patient Patient
	if err := json.Unmarshal(body, &patient); err != nil {
		c.log.Warn("hospitala decode failure", "id_masked", maskID(id))
		return nil, fmt.Errorf("%w: decode: %v", platform.ErrUpstream, err)
	}
	return &patient, nil
}

// ValidID reports whether id is alphanumeric and length-bounded (1–64).
func ValidID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func maskID(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}
