package duckdb

import (
	"database/sql"
	"fmt"

	_ "github.com/marcboeker/go-duckdb"
)

// Client wraps a DuckDB database connection.
type Client struct {
	db *sql.DB
}

// NewClient opens (or creates) a DuckDB database file.
// path can be ":memory:" for in-memory or a file path for persistent storage.
func NewClient(path string) (*Client, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("duckdb open: %w", err)
	}
	return &Client{db: db}, nil
}

// DB exposes the underlying *sql.DB for use by store implementations.
func (c *Client) DB() *sql.DB { return c.db }

// Close closes the underlying database connection.
func (c *Client) Close() error { return c.db.Close() }

// InitSchema creates the spans table and indexes if they don't exist.
func (c *Client) InitSchema() error {
	for _, stmt := range schemaStatements {
		if _, err := c.db.Exec(stmt); err != nil {
			return fmt.Errorf("init schema: %w", err)
		}
	}
	return nil
}

// schemaStatements are executed individually because DuckDB does not support
// multiple statements in a single Exec call.
var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS spans (
    trace_id         VARCHAR NOT NULL,
    span_id          VARCHAR NOT NULL,
    parent_span_id   VARCHAR,
    start_time       TIMESTAMP NOT NULL,
    end_time         TIMESTAMP NOT NULL,
    duration_ns      BIGINT NOT NULL,
    service_name     VARCHAR NOT NULL,
    action           VARCHAR,
    method           VARCHAR,
    path             VARCHAR,
    status_code      INTEGER,
    error            VARCHAR,
    tenant_id        VARCHAR,
    org_id           VARCHAR,
    user_id          VARCHAR,
    mem_alloc_kb     DOUBLE,
    goroutines       INTEGER,
    metadata         VARCHAR,
    request_headers  VARCHAR,
    request_body     VARCHAR,
    response_body    VARCHAR
)`,
	`CREATE INDEX IF NOT EXISTS idx_spans_trace_id ON spans(trace_id)`,
	`CREATE INDEX IF NOT EXISTS idx_spans_service ON spans(service_name, start_time)`,
	`CREATE INDEX IF NOT EXISTS idx_spans_tenant ON spans(tenant_id, start_time)`,
}
