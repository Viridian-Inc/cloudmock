// Package tenantscope provides wrapper implementations of TraceReader and
// RequestReader that enforce tenant-level visibility boundaries. When a
// request is made by a user with a tenant_id, only data belonging to that
// tenant is returned. Admin users and unauthenticated requests (auth
// disabled) bypass scoping entirely.
package tenantscope

import (
	"context"

	"github.com/neureaux/cloudmock/pkg/auth"
	"github.com/neureaux/cloudmock/pkg/dataplane"
)

// ---------------------------------------------------------------------------
// ScopedTraceReader
// ---------------------------------------------------------------------------

// ScopedTraceReader wraps a TraceReader and injects tenant filtering so that
// non-admin users only see traces belonging to their tenant.
type ScopedTraceReader struct {
	inner dataplane.TraceReader
}

// NewTraceReader returns a tenant-scoped wrapper around the given TraceReader.
func NewTraceReader(inner dataplane.TraceReader) *ScopedTraceReader {
	return &ScopedTraceReader{inner: inner}
}

// Get retrieves a trace by ID. For tenant-scoped users the trace is only
// returned when at least one span in the tree carries the user's tenant_id
// in its Metadata; otherwise ErrNotFound is returned.
func (r *ScopedTraceReader) Get(ctx context.Context, traceID string) (*dataplane.TraceContext, error) {
	user := auth.UserFromContext(ctx)
	if user == nil || user.TenantID == "" || user.Role == auth.RoleAdmin {
		return r.inner.Get(ctx, traceID)
	}

	tc, err := r.inner.Get(ctx, traceID)
	if err != nil {
		return nil, err
	}
	if !traceHasTenant(tc, user.TenantID) {
		return nil, dataplane.ErrNotFound
	}
	return tc, nil
}

// Search delegates to the inner reader after injecting TenantID into the
// filter for tenant-scoped users.
func (r *ScopedTraceReader) Search(ctx context.Context, filter dataplane.TraceFilter) ([]dataplane.TraceSummary, error) {
	user := auth.UserFromContext(ctx)
	if user != nil && user.TenantID != "" && user.Role != auth.RoleAdmin {
		filter.TenantID = user.TenantID
	}
	return r.inner.Search(ctx, filter)
}

// Timeline returns the timeline for a trace after verifying the caller has
// access via Get.
func (r *ScopedTraceReader) Timeline(ctx context.Context, traceID string) ([]dataplane.TimelineSpan, error) {
	if _, err := r.Get(ctx, traceID); err != nil {
		return nil, err
	}
	return r.inner.Timeline(ctx, traceID)
}

// ---------------------------------------------------------------------------
// ScopedRequestReader
// ---------------------------------------------------------------------------

// ScopedRequestReader wraps a RequestReader and injects tenant filtering so
// that non-admin users only see requests belonging to their tenant.
type ScopedRequestReader struct {
	inner dataplane.RequestReader
}

// NewRequestReader returns a tenant-scoped wrapper around the given
// RequestReader.
func NewRequestReader(inner dataplane.RequestReader) *ScopedRequestReader {
	return &ScopedRequestReader{inner: inner}
}

// Query delegates to the inner reader after injecting TenantID into the
// filter for tenant-scoped users.
func (r *ScopedRequestReader) Query(ctx context.Context, filter dataplane.RequestFilter) ([]dataplane.RequestEntry, error) {
	user := auth.UserFromContext(ctx)
	if user != nil && user.TenantID != "" && user.Role != auth.RoleAdmin {
		filter.TenantID = user.TenantID
	}
	return r.inner.Query(ctx, filter)
}

// GetByID retrieves a request by ID. For tenant-scoped users the request is
// only returned when its TenantID matches the caller's; otherwise
// ErrNotFound is returned.
func (r *ScopedRequestReader) GetByID(ctx context.Context, id string) (*dataplane.RequestEntry, error) {
	user := auth.UserFromContext(ctx)
	entry, err := r.inner.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user != nil && user.TenantID != "" && user.Role != auth.RoleAdmin {
		if entry.TenantID != user.TenantID {
			return nil, dataplane.ErrNotFound
		}
	}
	return entry, nil
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// traceHasTenant recursively checks whether any node in the trace tree
// carries the given tenant_id in its Metadata map.
func traceHasTenant(tc *dataplane.TraceContext, tenantID string) bool {
	if tc == nil {
		return false
	}
	if tc.Metadata != nil && tc.Metadata["tenant_id"] == tenantID {
		return true
	}
	for _, child := range tc.Children {
		if traceHasTenant(child, tenantID) {
			return true
		}
	}
	return false
}
