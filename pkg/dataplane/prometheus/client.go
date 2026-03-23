package prometheus

import (
	"fmt"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/neureaux/cloudmock/pkg/config"
)

// Client wraps the Prometheus HTTP API.
type Client struct {
	api promv1.API
}

// NewClient creates a Client configured from cfg.
func NewClient(cfg config.PrometheusConfig) (*Client, error) {
	client, err := promapi.NewClient(promapi.Config{Address: cfg.URL})
	if err != nil {
		return nil, fmt.Errorf("prometheus client: %w", err)
	}
	return &Client{api: promv1.NewAPI(client)}, nil
}

// API returns the underlying Prometheus v1 API.
func (c *Client) API() promv1.API {
	return c.api
}
