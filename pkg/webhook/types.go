package webhook

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a webhook config does not exist.
var ErrNotFound = errors.New("webhook not found")

// Config holds the configuration for a single outbound webhook.
type Config struct {
	ID      string            `json:"id"`
	URL     string            `json:"url"`
	Type    string            `json:"type"`    // "slack", "pagerduty", "generic"
	Events  []string          `json:"events"`  // "incident.created", "incident.resolved"
	Headers map[string]string `json:"headers,omitempty"`
	Active  bool              `json:"active"`
}

// Store persists and retrieves webhook configs.
type Store interface {
	Save(ctx context.Context, cfg *Config) error
	Get(ctx context.Context, id string) (*Config, error)
	List(ctx context.Context) ([]Config, error)
	Delete(ctx context.Context, id string) error
	ListByEvent(ctx context.Context, event string) ([]Config, error)
}
