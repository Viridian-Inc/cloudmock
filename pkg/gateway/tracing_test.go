package gateway

import (
	"testing"
	"time"
)

func TestTraceStoreAddRoot(t *testing.T) {
	ts := NewTraceStore(10)
	trace := &TraceContext{TraceID: "t1", SpanID: "s1", Service: "dynamodb", Action: "PutItem",
		StartTime: time.Now(), DurationMs: 5}
	ts.Add(trace)

	got := ts.Get("t1")
	if got == nil {
		t.Fatal("expected trace t1")
	}
	if got.SpanID != "s1" {
		t.Errorf("expected span s1, got %s", got.SpanID)
	}
}

func TestTraceStoreMergeChild(t *testing.T) {
	ts := NewTraceStore(10)
	now := time.Now()

	// Add root span
	root := &TraceContext{TraceID: "t1", SpanID: "root", Service: "lambda", Action: "Invoke",
		StartTime: now, DurationMs: 50}
	ts.Add(root)

	// Add child span with ParentSpanID
	child := &TraceContext{TraceID: "t1", SpanID: "child1", ParentSpanID: "root",
		Service: "dynamodb", Action: "Query", StartTime: now.Add(5 * time.Millisecond), DurationMs: 10}
	ts.Add(child)

	got := ts.Get("t1")
	if got == nil {
		t.Fatal("expected trace t1")
	}
	if len(got.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(got.Children))
	}
	if got.Children[0].SpanID != "child1" {
		t.Errorf("expected child span child1, got %s", got.Children[0].SpanID)
	}
	if got.Children[0].Service != "dynamodb" {
		t.Errorf("expected dynamodb, got %s", got.Children[0].Service)
	}
}

func TestTraceStoreMergeNestedChild(t *testing.T) {
	ts := NewTraceStore(10)
	now := time.Now()

	root := &TraceContext{TraceID: "t1", SpanID: "root", Service: "bff", StartTime: now, DurationMs: 100}
	ts.Add(root)

	child := &TraceContext{TraceID: "t1", SpanID: "c1", ParentSpanID: "root",
		Service: "lambda", StartTime: now.Add(5 * time.Millisecond), DurationMs: 40}
	ts.Add(child)

	grandchild := &TraceContext{TraceID: "t1", SpanID: "gc1", ParentSpanID: "c1",
		Service: "dynamodb", StartTime: now.Add(10 * time.Millisecond), DurationMs: 15}
	ts.Add(grandchild)

	got := ts.Get("t1")
	if got == nil {
		t.Fatal("expected trace t1")
	}
	if len(got.Children) != 1 {
		t.Fatalf("root should have 1 child, got %d", len(got.Children))
	}
	if len(got.Children[0].Children) != 1 {
		t.Fatalf("child should have 1 grandchild, got %d", len(got.Children[0].Children))
	}
	if got.Children[0].Children[0].Service != "dynamodb" {
		t.Errorf("grandchild service should be dynamodb, got %s", got.Children[0].Children[0].Service)
	}
}

func TestTraceStoreTimeline(t *testing.T) {
	ts := NewTraceStore(10)
	now := time.Now()

	root := &TraceContext{TraceID: "t1", SpanID: "root", Service: "bff", StartTime: now, DurationMs: 50}
	ts.Add(root)

	child := &TraceContext{TraceID: "t1", SpanID: "c1", ParentSpanID: "root",
		Service: "dynamodb", StartTime: now.Add(5 * time.Millisecond), DurationMs: 20}
	ts.Add(child)

	timeline := ts.Timeline("t1")
	if len(timeline) != 2 {
		t.Fatalf("expected 2 timeline spans, got %d", len(timeline))
	}
	if timeline[0].Depth != 0 {
		t.Errorf("root depth should be 0, got %d", timeline[0].Depth)
	}
	if timeline[1].Depth != 1 {
		t.Errorf("child depth should be 1, got %d", timeline[1].Depth)
	}
}
