package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
	"github.com/Viridian-Inc/cloudmock/pkg/dataplane/memory"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

func TestTraceStore_Get_Found(t *testing.T) {
	gs := gateway.NewTraceStore(100)
	gs.Add(&gateway.TraceContext{
		TraceID:    "trace-1",
		SpanID:     "span-1",
		Service:    "dynamodb",
		Action:     "PutItem",
		Method:     "POST",
		Path:       "/",
		StartTime:  time.Now(),
		EndTime:    time.Now().Add(10 * time.Millisecond),
		Duration:   10 * time.Millisecond,
		DurationMs: 10,
		StatusCode: 200,
		Metadata:   map[string]string{"x-tenant-id": "t1"},
		Children: []*gateway.TraceContext{
			{
				TraceID:    "trace-1",
				SpanID:     "span-2",
				ParentSpanID: "span-1",
				Service:    "s3",
				Action:     "GetObject",
				DurationMs: 5,
				StatusCode: 200,
			},
		},
	})

	s := memory.NewTraceStore(gs)
	tc, err := s.Get(context.Background(), "trace-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.TraceID != "trace-1" {
		t.Errorf("got TraceID=%q, want %q", tc.TraceID, "trace-1")
	}
	if tc.Service != "dynamodb" {
		t.Errorf("got Service=%q, want %q", tc.Service, "dynamodb")
	}
	if len(tc.Children) != 1 {
		t.Fatalf("got %d children, want 1", len(tc.Children))
	}
	if tc.Children[0].SpanID != "span-2" {
		t.Errorf("child SpanID=%q, want %q", tc.Children[0].SpanID, "span-2")
	}
	if tc.Metadata["x-tenant-id"] != "t1" {
		t.Errorf("metadata missing x-tenant-id")
	}
}

func TestTraceStore_Get_NotFound(t *testing.T) {
	gs := gateway.NewTraceStore(100)
	s := memory.NewTraceStore(gs)

	_, err := s.Get(context.Background(), "nonexistent")
	if err != dataplane.ErrNotFound {
		t.Errorf("got err=%v, want ErrNotFound", err)
	}
}

func TestTraceStore_Search(t *testing.T) {
	gs := gateway.NewTraceStore(100)
	gs.Add(&gateway.TraceContext{
		TraceID:    "trace-a",
		SpanID:     "span-a",
		Service:    "dynamodb",
		Action:     "Query",
		StartTime:  time.Now(),
		DurationMs: 5,
		StatusCode: 200,
	})
	gs.Add(&gateway.TraceContext{
		TraceID:    "trace-b",
		SpanID:     "span-b",
		Service:    "s3",
		Action:     "GetObject",
		StartTime:  time.Now(),
		DurationMs: 8,
		StatusCode: 200,
	})

	s := memory.NewTraceStore(gs)

	// Filter by service.
	results, err := s.Search(context.Background(), dataplane.TraceFilter{
		Service: "dynamodb",
		Limit:   10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].TraceID != "trace-a" {
		t.Errorf("got TraceID=%q, want %q", results[0].TraceID, "trace-a")
	}
	if results[0].RootService != "dynamodb" {
		t.Errorf("got RootService=%q, want %q", results[0].RootService, "dynamodb")
	}
}

func TestTraceStore_Timeline(t *testing.T) {
	gs := gateway.NewTraceStore(100)
	now := time.Now()
	gs.Add(&gateway.TraceContext{
		TraceID:    "trace-t",
		SpanID:     "span-root",
		Service:    "api",
		Action:     "Handle",
		StartTime:  now,
		EndTime:    now.Add(20 * time.Millisecond),
		DurationMs: 20,
		StatusCode: 200,
		Children: []*gateway.TraceContext{
			{
				TraceID:      "trace-t",
				SpanID:       "span-child",
				ParentSpanID: "span-root",
				Service:      "dynamodb",
				Action:       "Query",
				StartTime:    now.Add(2 * time.Millisecond),
				EndTime:      now.Add(10 * time.Millisecond),
				DurationMs:   8,
				StatusCode:   200,
			},
		},
	})

	s := memory.NewTraceStore(gs)
	spans, err := s.Timeline(context.Background(), "trace-t")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spans) != 2 {
		t.Fatalf("got %d spans, want 2", len(spans))
	}
	if spans[0].SpanID != "span-root" {
		t.Errorf("first span=%q, want %q", spans[0].SpanID, "span-root")
	}
	if spans[1].SpanID != "span-child" {
		t.Errorf("second span=%q, want %q", spans[1].SpanID, "span-child")
	}
	if spans[1].Depth != 1 {
		t.Errorf("child depth=%d, want 1", spans[1].Depth)
	}
}

func TestTraceStore_Timeline_NotFound(t *testing.T) {
	gs := gateway.NewTraceStore(100)
	s := memory.NewTraceStore(gs)

	_, err := s.Timeline(context.Background(), "nonexistent")
	if err != dataplane.ErrNotFound {
		t.Errorf("got err=%v, want ErrNotFound", err)
	}
}

func TestTraceStore_WriteSpans(t *testing.T) {
	gs := gateway.NewTraceStore(100)
	s := memory.NewTraceStore(gs)

	err := s.WriteSpans(context.Background(), []*dataplane.Span{
		{
			TraceID:    "trace-w",
			SpanID:     "span-w",
			Service:    "lambda",
			Action:     "Invoke",
			StartTime:  time.Now(),
			EndTime:    time.Now().Add(5 * time.Millisecond),
			DurationNs: uint64(5 * time.Millisecond),
			StatusCode: 200,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tc, err := s.Get(context.Background(), "trace-w")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc.Service != "lambda" {
		t.Errorf("got Service=%q, want %q", tc.Service, "lambda")
	}
}
