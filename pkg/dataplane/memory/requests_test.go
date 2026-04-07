package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
	"github.com/Viridian-Inc/cloudmock/pkg/dataplane/memory"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

func TestRequestStore_Query(t *testing.T) {
	gl := gateway.NewRequestLog(100)
	gl.Add(gateway.RequestEntry{
		ID:        "req-1",
		Service:   "dynamodb",
		Action:    "PutItem",
		Method:    "POST",
		Path:      "/",
		StatusCode: 200,
		LatencyMs: 5,
		Timestamp: time.Now(),
	})
	gl.Add(gateway.RequestEntry{
		ID:        "req-2",
		Service:   "s3",
		Action:    "GetObject",
		Method:    "GET",
		Path:      "/bucket/key",
		StatusCode: 200,
		LatencyMs: 10,
		Timestamp: time.Now(),
	})

	s := memory.NewRequestStore(gl)
	results, err := s.Query(context.Background(), dataplane.RequestFilter{
		Service: "dynamodb",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].ID != "req-1" {
		t.Errorf("got ID=%q, want %q", results[0].ID, "req-1")
	}
	if results[0].Service != "dynamodb" {
		t.Errorf("got Service=%q, want %q", results[0].Service, "dynamodb")
	}
}

func TestRequestStore_Query_ErrorOnly(t *testing.T) {
	gl := gateway.NewRequestLog(100)
	gl.Add(gateway.RequestEntry{
		ID:         "req-ok",
		Service:    "s3",
		StatusCode: 200,
		Timestamp:  time.Now(),
	})
	gl.Add(gateway.RequestEntry{
		ID:         "req-err",
		Service:    "s3",
		StatusCode: 500,
		Error:      "internal error",
		Timestamp:  time.Now(),
	})

	s := memory.NewRequestStore(gl)
	results, err := s.Query(context.Background(), dataplane.RequestFilter{
		ErrorOnly: true,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].ID != "req-err" {
		t.Errorf("got ID=%q, want %q", results[0].ID, "req-err")
	}
}

func TestRequestStore_GetByID(t *testing.T) {
	gl := gateway.NewRequestLog(100)
	gl.Add(gateway.RequestEntry{
		ID:        "req-42",
		Service:   "lambda",
		Action:    "Invoke",
		Method:    "POST",
		StatusCode: 200,
		LatencyMs: 15,
		Timestamp: time.Now(),
		RequestHeaders: map[string]string{
			"X-Tenant-Id":     "tenant-1",
			"X-Enterprise-Id": "org-1",
			"X-User-Id":       "user-1",
		},
	})

	s := memory.NewRequestStore(gl)

	entry, err := s.GetByID(context.Background(), "req-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != "req-42" {
		t.Errorf("got ID=%q, want %q", entry.ID, "req-42")
	}
	if entry.TenantID != "tenant-1" {
		t.Errorf("got TenantID=%q, want %q", entry.TenantID, "tenant-1")
	}
	if entry.OrgID != "org-1" {
		t.Errorf("got OrgID=%q, want %q", entry.OrgID, "org-1")
	}
	if entry.UserID != "user-1" {
		t.Errorf("got UserID=%q, want %q", entry.UserID, "user-1")
	}
}

func TestRequestStore_GetByID_NotFound(t *testing.T) {
	gl := gateway.NewRequestLog(100)
	s := memory.NewRequestStore(gl)

	_, err := s.GetByID(context.Background(), "nonexistent")
	if err != dataplane.ErrNotFound {
		t.Errorf("got err=%v, want ErrNotFound", err)
	}
}

func TestRequestStore_Write(t *testing.T) {
	gl := gateway.NewRequestLog(100)
	s := memory.NewRequestStore(gl)

	err := s.Write(context.Background(), dataplane.RequestEntry{
		ID:        "req-w1",
		Service:   "sqs",
		Action:    "SendMessage",
		Method:    "POST",
		StatusCode: 200,
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entry, err := s.GetByID(context.Background(), "req-w1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.Service != "sqs" {
		t.Errorf("got Service=%q, want %q", entry.Service, "sqs")
	}
}
