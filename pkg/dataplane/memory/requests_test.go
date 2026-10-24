package memory_test

import (
	"context"
	"maps"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/dataplane"
	"github.com/Viridian-Inc/cloudmock/pkg/dataplane/memory"
	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

func TestRequestStore_Query(t *testing.T) {
	gl := gateway.NewRequestLog(100)
	gl.Add(gateway.RequestEntry{
		ID:         "req-1",
		Service:    "dynamodb",
		Action:     "PutItem",
		Method:     "POST",
		Path:       "/",
		StatusCode: 200,
		LatencyMs:  5,
		Timestamp:  time.Now(),
	})
	gl.Add(gateway.RequestEntry{
		ID:         "req-2",
		Service:    "s3",
		Action:     "GetObject",
		Method:     "GET",
		Path:       "/bucket/key",
		StatusCode: 200,
		LatencyMs:  10,
		Timestamp:  time.Now(),
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
		ID:         "req-42",
		Service:    "lambda",
		Action:     "Invoke",
		Method:     "POST",
		StatusCode: 200,
		LatencyMs:  15,
		Timestamp:  time.Now(),
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
		ID:         "req-w1",
		Service:    "sqs",
		Action:     "SendMessage",
		Method:     "POST",
		StatusCode: 200,
		Timestamp:  time.Now(),
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

func assertRequestScope(t *testing.T, entry *dataplane.RequestEntry, tenant, org, user string) {
	t.Helper()
	if entry == nil {
		t.Fatal("request entry missing")
	}
	got, want := [3]string{entry.TenantID, entry.OrgID, entry.UserID}, [3]string{tenant, org, user}
	if got != want {
		t.Errorf("scope for %s=%q, want %q", entry.ID, got, want)
	}
}

func assertScopedRequestIDs(t *testing.T, reader dataplane.RequestReader, filter dataplane.RequestFilter, want ...string) {
	t.Helper()
	entries, err := reader.Query(context.Background(), filter)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	got := make([]string, 0, len(entries))
	for _, entry := range entries {
		got = append(got, entry.ID)
	}
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) || (len(got) > 0 && !reflect.DeepEqual(got, want)) {
		t.Errorf("filter %+v returned %v, want %v", filter, got, want)
	}
}

func TestRequestStore_TypedScopeRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers map[string]string
	}{{"nil-headers", nil}, {"empty-headers", map[string]string{}}} {
		t.Run(tc.name, func(t *testing.T) {
			store := memory.NewRequestStore(gateway.NewRequestLog(10))
			var writer dataplane.RequestWriter = store
			var reader dataplane.RequestReader = store
			entry := dataplane.RequestEntry{
				ID: "typed", TenantID: "tenant-a", OrgID: "org-a", UserID: "user-a",
				Service: "s3", Action: "PutObject", RequestHeaders: tc.headers,
			}
			before := maps.Clone(tc.headers)
			if err := writer.Write(context.Background(), entry); err != nil {
				t.Fatal(err)
			}
			got, err := reader.GetByID(context.Background(), entry.ID)
			if err != nil {
				t.Fatal(err)
			}
			assertRequestScope(t, got, "tenant-a", "org-a", "user-a")
			results, err := reader.Query(context.Background(), dataplane.RequestFilter{Service: "s3"})
			if err != nil || len(results) != 1 {
				t.Fatalf("unscoped query: entries=%v err=%v", results, err)
			}
			assertRequestScope(t, &results[0], "tenant-a", "org-a", "user-a")
			if !reflect.DeepEqual(tc.headers, before) {
				t.Errorf("Write mutated caller headers: got %v want %v", tc.headers, before)
			}
		})
	}
}

func TestRequestStore_TypedScopeFilters(t *testing.T) {
	store := memory.NewRequestStore(gateway.NewRequestLog(20))
	var writer dataplane.RequestWriter = store
	var reader dataplane.RequestReader = store
	for _, entry := range []dataplane.RequestEntry{
		{ID: "target", TenantID: "tenant-a", OrgID: "org-a", UserID: "user-a"},
		{ID: "other-tenant", TenantID: "tenant-b", OrgID: "org-a", UserID: "user-a"},
		{ID: "other-org", TenantID: "tenant-a", OrgID: "org-b", UserID: "user-a"},
		{ID: "other-user", TenantID: "tenant-a", OrgID: "org-a", UserID: "user-b"},
		{ID: "other-all", TenantID: "tenant-b", OrgID: "org-b", UserID: "user-b"},
		{ID: "unscoped"},
	} {
		if err := writer.Write(context.Background(), entry); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name   string
		filter dataplane.RequestFilter
		want   []string
	}{
		{"tenant", dataplane.RequestFilter{TenantID: "tenant-a"}, []string{"target", "other-org", "other-user"}},
		{"org", dataplane.RequestFilter{OrgID: "org-a"}, []string{"target", "other-tenant", "other-user"}},
		{"user", dataplane.RequestFilter{UserID: "user-a"}, []string{"target", "other-tenant", "other-org"}},
		{"tenant-org", dataplane.RequestFilter{TenantID: "tenant-a", OrgID: "org-a"}, []string{"target", "other-user"}},
		{"tenant-user", dataplane.RequestFilter{TenantID: "tenant-a", UserID: "user-a"}, []string{"target", "other-org"}},
		{"org-user", dataplane.RequestFilter{OrgID: "org-a", UserID: "user-a"}, []string{"target", "other-tenant"}},
		{"all", dataplane.RequestFilter{TenantID: "tenant-a", OrgID: "org-a", UserID: "user-a"}, []string{"target"}},
		{"missing", dataplane.RequestFilter{TenantID: "missing"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertScopedRequestIDs(t, reader, tc.filter, tc.want...)
		})
	}
}

func TestRequestStore_LegacyScopeHeaders(t *testing.T) {
	for _, direct := range []bool{false, true} {
		name := "adapter-write"
		if direct {
			name = "gateway-write"
		}
		t.Run(name, func(t *testing.T) {
			log := gateway.NewRequestLog(10)
			store := memory.NewRequestStore(log)
			var writer dataplane.RequestWriter = store
			var reader dataplane.RequestReader = store
			headers := map[string]string{"X-Tenant-Id": "tenant-legacy", "X-Enterprise-Id": "org-legacy", "X-User-Id": "user-legacy"}
			if direct {
				log.Add(gateway.RequestEntry{ID: "legacy", RequestHeaders: headers})
			} else if err := writer.Write(context.Background(), dataplane.RequestEntry{ID: "legacy", RequestHeaders: headers}); err != nil {
				t.Fatal(err)
			}
			log.Add(gateway.RequestEntry{ID: "other", RequestHeaders: map[string]string{"X-Tenant-Id": "other", "X-Enterprise-Id": "other", "X-User-Id": "other"}})
			got, err := reader.GetByID(context.Background(), "legacy")
			if err != nil {
				t.Fatal(err)
			}
			assertRequestScope(t, got, "tenant-legacy", "org-legacy", "user-legacy")
			for _, filter := range []dataplane.RequestFilter{
				{TenantID: "tenant-legacy"}, {OrgID: "org-legacy"}, {UserID: "user-legacy"},
				{TenantID: "tenant-legacy", OrgID: "org-legacy", UserID: "user-legacy"},
			} {
				assertScopedRequestIDs(t, reader, filter, "legacy")
			}
		})
	}
}

func TestRequestStore_TypedScopePrecedence(t *testing.T) {
	for _, tc := range []struct {
		name  string
		typed [3]string
	}{
		{"tenant-only", [3]string{"tenant-typed", "", ""}},
		{"org-only", [3]string{"", "org-typed", ""}},
		{"user-only", [3]string{"", "", "user-typed"}},
		{"all-typed", [3]string{"tenant-typed", "org-typed", "user-typed"}},
		{"blank-typed-fallback", [3]string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := memory.NewRequestStore(gateway.NewRequestLog(10))
			var writer dataplane.RequestWriter = store
			var reader dataplane.RequestReader = store
			legacy := [3]string{"tenant-header", "org-header", "user-header"}
			headers := map[string]string{
				"X-Tenant-Id": legacy[0], "X-Enterprise-Id": legacy[1], "X-User-Id": legacy[2], "X-Custom": "preserve-me",
			}
			before := maps.Clone(headers)
			entry := dataplane.RequestEntry{ID: "mixed", TenantID: tc.typed[0], OrgID: tc.typed[1], UserID: tc.typed[2], RequestHeaders: headers}
			if err := writer.Write(context.Background(), entry); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(headers, before) {
				t.Errorf("Write mutated caller headers: got %v want %v", headers, before)
			}
			want := legacy
			for i, typed := range tc.typed {
				if typed != "" {
					want[i] = typed
				}
			}
			got, err := reader.GetByID(context.Background(), entry.ID)
			if err != nil {
				t.Fatal(err)
			}
			assertRequestScope(t, got, want[0], want[1], want[2])
			if got.RequestHeaders["X-Custom"] != "preserve-me" {
				t.Error("unrelated request header was lost")
			}
			assertScopedRequestIDs(t, reader, dataplane.RequestFilter{TenantID: want[0], OrgID: want[1], UserID: want[2]}, "mixed")
			conflictingFilters := []dataplane.RequestFilter{{TenantID: legacy[0]}, {OrgID: legacy[1]}, {UserID: legacy[2]}}
			for i, typed := range tc.typed {
				if typed != "" {
					assertScopedRequestIDs(t, reader, conflictingFilters[i])
				}
			}
		})
	}
}

func TestRequestStore_ScopeHeaderOwnership(t *testing.T) {
	store := memory.NewRequestStore(gateway.NewRequestLog(10))
	var writer dataplane.RequestWriter = store
	var reader dataplane.RequestReader = store
	headers := map[string]string{
		"X-Tenant-Id": "legacy-tenant", "X-Enterprise-Id": "legacy-org", "X-User-Id": "legacy-user",
	}
	before := maps.Clone(headers)
	if err := writer.Write(context.Background(), dataplane.RequestEntry{
		ID: "owned", TenantID: "tenant-a", OrgID: "org-a", UserID: "user-a", RequestHeaders: headers,
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(headers, before) {
		t.Errorf("Write mutated caller headers: got %v want %v", headers, before)
	}
	mutate := func(headers map[string]string) {
		for _, key := range []string{"X-Tenant-Id", "X-Enterprise-Id", "X-User-Id"} {
			headers[key] = "mutated"
		}
	}
	verify := func() *dataplane.RequestEntry {
		t.Helper()
		entry, err := reader.GetByID(context.Background(), "owned")
		if err != nil {
			t.Fatal(err)
		}
		assertRequestScope(t, entry, "tenant-a", "org-a", "user-a")
		assertScopedRequestIDs(t, reader, dataplane.RequestFilter{TenantID: "tenant-a", OrgID: "org-a", UserID: "user-a"}, "owned")
		assertScopedRequestIDs(t, reader, dataplane.RequestFilter{TenantID: "mutated"})
		return entry
	}
	// A caller retains ownership of both its input and returned record maps.
	mutate(headers)
	fromGet := verify()
	mutate(fromGet.RequestHeaders)
	verify()
	fromQuery, err := reader.Query(context.Background(), dataplane.RequestFilter{})
	if err != nil || len(fromQuery) != 1 {
		t.Fatalf("Query: entries=%v err=%v", fromQuery, err)
	}
	assertRequestScope(t, &fromQuery[0], "tenant-a", "org-a", "user-a")
	mutate(fromQuery[0].RequestHeaders)
	verify()
}
