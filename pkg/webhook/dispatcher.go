package webhook

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Dispatcher fires outbound HTTP webhooks for incident events.
type Dispatcher struct {
	store  Store
	client *http.Client
}

// NewDispatcher creates a Dispatcher backed by the given store.
// A default HTTP client with a 10-second timeout is used.
func NewDispatcher(store Store) *Dispatcher {
	return &Dispatcher{
		store: store,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Store returns the underlying webhook store.
func (d *Dispatcher) Store() Store { return d.store }

// Fire sends the payload to all active webhooks registered for the given event.
// It queries the store for matching configs, formats the payload per webhook
// type, and POSTs to each URL. Errors are logged but do not fail the caller
// (best-effort delivery).
func (d *Dispatcher) Fire(ctx context.Context, event string, payload any) error {
	configs, err := d.store.ListByEvent(ctx, event)
	if err != nil {
		log.Printf("webhook: list by event %q: %v", event, err)
		return nil // best-effort
	}

	for _, cfg := range configs {
		if !cfg.Active {
			continue
		}
		if err := d.send(ctx, cfg, event, payload); err != nil {
			log.Printf("webhook: send to %s (id=%s): %v", cfg.URL, cfg.ID, err)
		}
	}
	return nil
}

// FireToConfig sends a payload directly to a single webhook config, bypassing
// the store lookup. Useful for test deliveries.
func (d *Dispatcher) FireToConfig(ctx context.Context, cfg Config, event string, payload any) error {
	return d.send(ctx, cfg, event, payload)
}

// send formats and POSTs the payload to a single webhook config.
func (d *Dispatcher) send(ctx context.Context, cfg Config, event string, payload any) error {
	body, err := format(cfg.Type, event, payload)
	if err != nil {
		return fmt.Errorf("format: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}
	return nil
}

// format dispatches to the appropriate formatter based on webhook type.
func format(typ, event string, payload any) ([]byte, error) {
	switch typ {
	case "slack":
		return FormatSlack(event, payload)
	case "pagerduty":
		return FormatPagerDuty(event, payload)
	default:
		return FormatGeneric(event, payload)
	}
}
