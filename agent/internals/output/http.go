package output

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"time"

	"featuretrace.io/agent/internals/model"
)

// HTTPExporter sends record batches to the FeatureTrace backend via
// POST /ingest/logs with optional gzip compression and exponential-backoff
// retry.
type HTTPExporter struct {
	Endpoint    string
	Timeout     time.Duration
	MaxRetries  int
	Compression bool

	client *http.Client
}

// NewHTTPExporter creates an exporter targeting the given endpoint.
func NewHTTPExporter(endpoint string, timeout time.Duration, maxRetries int, compression bool) *HTTPExporter {
	return &HTTPExporter{
		Endpoint:    endpoint + "/ingest/logs",
		Timeout:     timeout,
		MaxRetries:  maxRetries,
		Compression: compression,
		client:      &http.Client{Timeout: timeout},
	}
}

// Send transmits a batch with retries and backpressure.
func (h *HTTPExporter) Send(ctx context.Context, batch []model.Record) error {
	body, err := h.encode(batch)
	if err != nil {
		return fmt.Errorf("encoding batch: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= h.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * 500 * time.Millisecond
			log.Printf("[exporter] retry %d/%d in %s", attempt, h.MaxRetries, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		lastErr = h.doPost(ctx, body)
		if lastErr == nil {
			log.Printf("[exporter] sent batch of %d records", len(batch))
			return nil
		}

		log.Printf("[exporter] attempt %d failed: %v", attempt+1, lastErr)
	}

	// All retries exhausted — drop batch (backpressure: never crash)
	log.Printf("[exporter] dropping batch of %d records after %d retries: %v",
		len(batch), h.MaxRetries, lastErr)
	return lastErr
}

// encode serialises the batch, optionally compressing it.
func (h *HTTPExporter) encode(batch []model.Record) ([]byte, error) {
	payload, err := json.Marshal(batch)
	if err != nil {
		return nil, err
	}

	if !h.Compression {
		return payload, nil
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(payload); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// doPost performs a single HTTP POST.
func (h *HTTPExporter) doPost(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if h.Compression {
		req.Header.Set("Content-Encoding", "gzip")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body) // drain body for connection reuse

	if resp.StatusCode >= 400 {
		return fmt.Errorf("backend returned %d", resp.StatusCode)
	}
	return nil
}

