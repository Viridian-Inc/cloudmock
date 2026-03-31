package tenant

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// MemoryStore is an in-memory Store implementation for testing.
type MemoryStore struct {
	mu      sync.RWMutex
	tenants map[string]Tenant // keyed by ID
	usage   map[string]UsageRecord
}

// NewMemoryStore creates a new empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tenants: make(map[string]Tenant),
		usage:   make(map[string]UsageRecord),
	}
}

func (m *MemoryStore) Create(_ context.Context, t *Tenant) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	// Check uniqueness constraints.
	for _, existing := range m.tenants {
		if existing.Slug == t.Slug {
			return errDuplicate("slug", t.Slug)
		}
		if existing.ClerkOrgID == t.ClerkOrgID {
			return errDuplicate("clerk_org_id", t.ClerkOrgID)
		}
	}

	m.tenants[t.ID] = *t
	return nil
}

func (m *MemoryStore) Get(_ context.Context, id string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.tenants[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &t, nil
}

func (m *MemoryStore) GetBySlug(_ context.Context, slug string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.tenants {
		if t.Slug == slug {
			return &t, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) GetByClerkOrgID(_ context.Context, clerkOrgID string) (*Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.tenants {
		if t.ClerkOrgID == clerkOrgID {
			return &t, nil
		}
	}
	return nil, ErrNotFound
}

func (m *MemoryStore) List(_ context.Context) ([]Tenant, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Tenant, 0, len(m.tenants))
	for _, t := range m.tenants {
		result = append(result, t)
	}
	return result, nil
}

func (m *MemoryStore) Update(_ context.Context, t *Tenant) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tenants[t.ID]; !ok {
		return ErrNotFound
	}
	t.UpdatedAt = time.Now()
	m.tenants[t.ID] = *t
	return nil
}

func (m *MemoryStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tenants[id]; !ok {
		return ErrNotFound
	}
	delete(m.tenants, id)
	return nil
}

func (m *MemoryStore) IncrementRequestCount(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tenants[id]
	if !ok {
		return ErrNotFound
	}
	t.RequestCount++
	t.UpdatedAt = time.Now()
	m.tenants[id] = t
	return nil
}

func (m *MemoryStore) RecordUsage(_ context.Context, record *UsageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	record.CreatedAt = time.Now()
	m.usage[record.ID] = *record
	return nil
}

func (m *MemoryStore) GetUnreportedUsage(_ context.Context) ([]UsageRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []UsageRecord
	for _, u := range m.usage {
		if !u.ReportedToStripe {
			result = append(result, u)
		}
	}
	return result, nil
}

func (m *MemoryStore) MarkUsageReported(_ context.Context, recordID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, ok := m.usage[recordID]
	if !ok {
		return ErrNotFound
	}
	u.ReportedToStripe = true
	m.usage[recordID] = u
	return nil
}

// errDuplicate returns a formatted duplicate-key error.
func errDuplicate(field, value string) error {
	return &DuplicateError{Field: field, Value: value}
}

// DuplicateError is returned when a uniqueness constraint is violated.
type DuplicateError struct {
	Field string
	Value string
}

func (e *DuplicateError) Error() string {
	return "tenant: duplicate " + e.Field + " " + e.Value
}

// Compile-time interface check.
var _ Store = (*MemoryStore)(nil)
