package tenantscope

import (
	"context"
	"testing"

	"github.com/neureaux/cloudmock/pkg/auth"
	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// ---------------------------------------------------------------------------
// mock TraceReader
// ---------------------------------------------------------------------------

type mockTraceReader struct {
	getFunc      func(ctx context.Context, traceID string) (*dataplane.TraceContext, error)
	searchFunc   func(ctx context.Context, filter dataplane.TraceFilter) ([]dataplane.TraceSummary, error)
	timelineFunc func(ctx context.Context, traceID string) ([]dataplane.TimelineSpan, error)
}

func (m *mockTraceReader) Get(ctx context.Context, traceID string) (*dataplane.TraceContext, error) {
	return m.getFunc(ctx, traceID)
}

func (m *mockTraceReader) Search(ctx context.Context, filter dataplane.TraceFilter) ([]dataplane.TraceSummary, error) {
	return m.searchFunc(ctx, filter)
}

func (m *mockTraceReader) Timeline(ctx context.Context, traceID string) ([]dataplane.TimelineSpan, error) {
	return m.timelineFunc(ctx, traceID)
}

// ---------------------------------------------------------------------------
// mock RequestReader
// ---------------------------------------------------------------------------

type mockRequestReader struct {
	queryFunc   func(ctx context.Context, filter dataplane.RequestFilter) ([]dataplane.RequestEntry, error)
	getByIDFunc func(ctx context.Context, id string) (*dataplane.RequestEntry, error)
}

func (m *mockRequestReader) Query(ctx context.Context, filter dataplane.RequestFilter) ([]dataplane.RequestEntry, error) {
	return m.queryFunc(ctx, filter)
}

func (m *mockRequestReader) GetByID(ctx context.Context, id string) (*dataplane.RequestEntry, error) {
	return m.getByIDFunc(ctx, id)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func traceWithTenant(tenantID string) *dataplane.TraceContext {
	return &dataplane.TraceContext{
		TraceID: "trace-1",
		Metadata: map[string]string{
			"tenant_id": tenantID,
		},
	}
}

func traceWithChildTenant(tenantID string) *dataplane.TraceContext {
	return &dataplane.TraceContext{
		TraceID:  "trace-2",
		Metadata: map[string]string{},
		Children: []*dataplane.TraceContext{
			{
				TraceID: "trace-2",
				Metadata: map[string]string{
					"tenant_id": tenantID,
				},
			},
		},
	}
}

func ctxWithUser(role, tenantID string) context.Context {
	return auth.ContextWithUser(context.Background(), &auth.User{
		ID:       "u1",
		Role:     role,
		TenantID: tenantID,
	})
}

// ---------------------------------------------------------------------------
// ScopedTraceReader tests
// ---------------------------------------------------------------------------

func TestScopedTraceReader_AdminBypass(t *testing.T) {
	inner := &mockTraceReader{
		getFunc: func(_ context.Context, _ string) (*dataplane.TraceContext, error) {
			return traceWithTenant("other-tenant"), nil
		},
		searchFunc: func(_ context.Context, f dataplane.TraceFilter) ([]dataplane.TraceSummary, error) {
			if f.TenantID != "" {
				t.Errorf("expected empty TenantID for admin, got %q", f.TenantID)
			}
			return nil, nil
		},
	}

	r := NewTraceReader(inner)
	ctx := ctxWithUser(auth.RoleAdmin, "t1")

	// Get should pass through without tenant check.
	tc, err := r.Get(ctx, "trace-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc == nil {
		t.Fatal("expected trace, got nil")
	}

	// Search should not inject TenantID filter.
	_, err = r.Search(ctx, dataplane.TraceFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScopedTraceReader_TenantFiltered(t *testing.T) {
	var capturedFilter dataplane.TraceFilter
	inner := &mockTraceReader{
		searchFunc: func(_ context.Context, f dataplane.TraceFilter) ([]dataplane.TraceSummary, error) {
			capturedFilter = f
			return []dataplane.TraceSummary{{TraceID: "t1"}}, nil
		},
	}

	r := NewTraceReader(inner)
	ctx := ctxWithUser(auth.RoleViewer, "tenant-abc")

	results, err := r.Search(ctx, dataplane.TraceFilter{Service: "s3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilter.TenantID != "tenant-abc" {
		t.Errorf("expected TenantID=tenant-abc, got %q", capturedFilter.TenantID)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestScopedTraceReader_GetDenied(t *testing.T) {
	inner := &mockTraceReader{
		getFunc: func(_ context.Context, _ string) (*dataplane.TraceContext, error) {
			return traceWithTenant("other-tenant"), nil
		},
	}

	r := NewTraceReader(inner)
	ctx := ctxWithUser(auth.RoleViewer, "my-tenant")

	_, err := r.Get(ctx, "trace-1")
	if err != dataplane.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestScopedTraceReader_GetAllowed(t *testing.T) {
	inner := &mockTraceReader{
		getFunc: func(_ context.Context, _ string) (*dataplane.TraceContext, error) {
			return traceWithTenant("my-tenant"), nil
		},
	}

	r := NewTraceReader(inner)
	ctx := ctxWithUser(auth.RoleViewer, "my-tenant")

	tc, err := r.Get(ctx, "trace-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc == nil {
		t.Fatal("expected trace, got nil")
	}
}

func TestScopedTraceReader_GetAllowedViaChild(t *testing.T) {
	inner := &mockTraceReader{
		getFunc: func(_ context.Context, _ string) (*dataplane.TraceContext, error) {
			return traceWithChildTenant("my-tenant"), nil
		},
	}

	r := NewTraceReader(inner)
	ctx := ctxWithUser(auth.RoleViewer, "my-tenant")

	tc, err := r.Get(ctx, "trace-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc == nil {
		t.Fatal("expected trace, got nil")
	}
}

func TestScopedTraceReader_NoAuth(t *testing.T) {
	inner := &mockTraceReader{
		getFunc: func(_ context.Context, _ string) (*dataplane.TraceContext, error) {
			return traceWithTenant("any"), nil
		},
		searchFunc: func(_ context.Context, f dataplane.TraceFilter) ([]dataplane.TraceSummary, error) {
			if f.TenantID != "" {
				t.Errorf("expected empty TenantID with no auth, got %q", f.TenantID)
			}
			return nil, nil
		},
	}

	r := NewTraceReader(inner)
	ctx := context.Background() // no user

	tc, err := r.Get(ctx, "trace-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tc == nil {
		t.Fatal("expected trace, got nil")
	}

	_, err = r.Search(ctx, dataplane.TraceFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScopedTraceReader_Timeline(t *testing.T) {
	inner := &mockTraceReader{
		getFunc: func(_ context.Context, _ string) (*dataplane.TraceContext, error) {
			return traceWithTenant("my-tenant"), nil
		},
		timelineFunc: func(_ context.Context, _ string) ([]dataplane.TimelineSpan, error) {
			return []dataplane.TimelineSpan{{SpanID: "s1"}}, nil
		},
	}

	r := NewTraceReader(inner)
	ctx := ctxWithUser(auth.RoleViewer, "my-tenant")

	spans, err := r.Timeline(ctx, "trace-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(spans) != 1 {
		t.Errorf("expected 1 span, got %d", len(spans))
	}
}

func TestScopedTraceReader_TimelineDenied(t *testing.T) {
	inner := &mockTraceReader{
		getFunc: func(_ context.Context, _ string) (*dataplane.TraceContext, error) {
			return traceWithTenant("other-tenant"), nil
		},
	}

	r := NewTraceReader(inner)
	ctx := ctxWithUser(auth.RoleViewer, "my-tenant")

	_, err := r.Timeline(ctx, "trace-1")
	if err != dataplane.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ScopedRequestReader tests
// ---------------------------------------------------------------------------

func TestScopedRequestReader_TenantFiltered(t *testing.T) {
	var capturedFilter dataplane.RequestFilter
	inner := &mockRequestReader{
		queryFunc: func(_ context.Context, f dataplane.RequestFilter) ([]dataplane.RequestEntry, error) {
			capturedFilter = f
			return []dataplane.RequestEntry{{ID: "r1"}}, nil
		},
	}

	r := NewRequestReader(inner)
	ctx := ctxWithUser(auth.RoleViewer, "tenant-xyz")

	results, err := r.Query(ctx, dataplane.RequestFilter{Service: "sqs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilter.TenantID != "tenant-xyz" {
		t.Errorf("expected TenantID=tenant-xyz, got %q", capturedFilter.TenantID)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestScopedRequestReader_GetByIDDenied(t *testing.T) {
	inner := &mockRequestReader{
		getByIDFunc: func(_ context.Context, _ string) (*dataplane.RequestEntry, error) {
			return &dataplane.RequestEntry{ID: "r1", TenantID: "other-tenant"}, nil
		},
	}

	r := NewRequestReader(inner)
	ctx := ctxWithUser(auth.RoleViewer, "my-tenant")

	_, err := r.GetByID(ctx, "r1")
	if err != dataplane.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestScopedRequestReader_GetByIDAllowed(t *testing.T) {
	inner := &mockRequestReader{
		getByIDFunc: func(_ context.Context, _ string) (*dataplane.RequestEntry, error) {
			return &dataplane.RequestEntry{ID: "r1", TenantID: "my-tenant"}, nil
		},
	}

	r := NewRequestReader(inner)
	ctx := ctxWithUser(auth.RoleViewer, "my-tenant")

	entry, err := r.GetByID(ctx, "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry.ID != "r1" {
		t.Errorf("expected r1, got %s", entry.ID)
	}
}

func TestScopedRequestReader_AdminBypass(t *testing.T) {
	inner := &mockRequestReader{
		getByIDFunc: func(_ context.Context, _ string) (*dataplane.RequestEntry, error) {
			return &dataplane.RequestEntry{ID: "r1", TenantID: "other-tenant"}, nil
		},
		queryFunc: func(_ context.Context, f dataplane.RequestFilter) ([]dataplane.RequestEntry, error) {
			if f.TenantID != "" {
				t.Errorf("expected empty TenantID for admin, got %q", f.TenantID)
			}
			return nil, nil
		},
	}

	r := NewRequestReader(inner)
	ctx := ctxWithUser(auth.RoleAdmin, "t1")

	entry, err := r.GetByID(ctx, "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	_, err = r.Query(ctx, dataplane.RequestFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
