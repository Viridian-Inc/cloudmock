package clickhouse_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/neureaux/cloudmock/pkg/config"
	chstore "github.com/neureaux/cloudmock/pkg/dataplane/clickhouse"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS spans (
    trace_id         FixedString(32),
    span_id          FixedString(16),
    parent_span_id   FixedString(16),
    start_time       DateTime64(9, 'UTC'),
    end_time         DateTime64(9, 'UTC'),
    duration_ns      UInt64,
    service_name     LowCardinality(String),
    action           LowCardinality(String),
    method           LowCardinality(String),
    path             String,
    status_code      UInt16,
    error            String,
    tenant_id        String,
    org_id           String,
    user_id          String,
    mem_alloc_kb     Float64,
    goroutines       UInt32,
    metadata         Map(String, String),
    request_headers  Map(String, String),
    request_body     String,
    response_body    String,
    _date            Date DEFAULT toDate(start_time)
)
ENGINE = MergeTree()
PARTITION BY (tenant_id, toYYYYMM(_date))
ORDER BY (service_name, action, start_time, trace_id)
TTL _date + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;
`

// setupClickHouse starts a ClickHouse container, applies the schema, and
// returns a connected Client. The container is cleaned up when the test ends.
func setupClickHouse(t *testing.T, ctx context.Context) *chstore.Client {
	t.Helper()

	req := testcontainers.ContainerRequest{
		Image:        "clickhouse/clickhouse-server:24-alpine",
		ExposedPorts: []string{"9000/tcp"},
		WaitingFor:   wait.ForListeningPort("9000/tcp"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start clickhouse container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "9000")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	cfg := config.ClickHouseConfig{
		Endpoint: fmt.Sprintf("%s:%s", host, port.Port()),
		Database: "default",
	}

	client, err := chstore.NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create clickhouse client: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})

	// Apply schema.
	if err := client.Conn().Exec(ctx, schemaSQL); err != nil {
		t.Fatalf("failed to apply schema: %v", err)
	}

	return client
}
