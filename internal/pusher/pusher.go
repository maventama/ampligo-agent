package pusher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/niago-id/ampligo-agent/internal/collector"
)

type Pusher struct {
	IngestURL string
	APIKey    string
	client    *http.Client
}

func New(ingestURL, apiKey string) *Pusher {
	return &Pusher{
		IngestURL: ingestURL,
		APIKey:    apiKey,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

const maxAttempts = 3

// Push sends one snapshot to the ingest endpoint, retrying transient failures
// (network errors, 5xx) with a short backoff. 4xx responses (bad key, bad
// payload) are not retried since a retry would fail identically.
func (p *Pusher) Push(snap collector.Snapshot) error {
	body, err := json.Marshal(snap)
	if err != nil {
		return fmt.Errorf("encoding snapshot: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		req, err := http.NewRequest(http.MethodPost, p.IngestURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Ampligo-Key", p.APIKey)

		resp, err := p.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d: %w", attempt, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return fmt.Errorf("ingest rejected push: HTTP %d", resp.StatusCode)
		}
		lastErr = fmt.Errorf("attempt %d: HTTP %d", attempt, resp.StatusCode)
	}

	return fmt.Errorf("giving up after %d attempts: %w", maxAttempts, lastErr)
}
