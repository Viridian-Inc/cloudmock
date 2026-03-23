package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/neureaux/cloudmock/pkg/config"
)

// Client wraps a ClickHouse connection.
type Client struct {
	conn driver.Conn
}

// NewClient opens a ClickHouse connection using the provided config, pings it,
// and returns a ready-to-use Client.
func NewClient(ctx context.Context, cfg config.ClickHouseConfig) (*Client, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{cfg.Endpoint},
		Auth: clickhouse.Auth{Database: cfg.Database},
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse open: %w", err)
	}
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}
	return &Client{conn: conn}, nil
}

// Close closes the underlying connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Conn exposes the underlying ClickHouse connection for use by store
// implementations.
func (c *Client) Conn() driver.Conn {
	return c.conn
}
