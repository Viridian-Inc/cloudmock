package iac

import (
	"errors"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// --- test doubles ---------------------------------------------------------

// fakeDynamo implements service.Service plus the GetTableNames and
// TableKeySchema structural interfaces that diffDynamo relies on.
type fakeDynamo struct {
	tables map[string]fakeTable
}

type fakeTable struct {
	hashKey  string
	rangeKey string
	gsis     map[string][2]string
}

func (f *fakeDynamo) Name() string              { return "dynamodb" }
func (f *fakeDynamo) Actions() []service.Action { return nil }
func (f *fakeDynamo) HealthCheck() error        { return nil }
func (f *fakeDynamo) HandleRequest(*service.RequestContext) (*service.Response, error) {
	return nil, nil
}

func (f *fakeDynamo) GetTableNames() []string {
	names := make([]string, 0, len(f.tables))
	for n := range f.tables {
		names = append(names, n)
	}
	return names
}

func (f *fakeDynamo) TableKeySchema(name string) (string, string, map[string][2]string, bool) {
	t, ok := f.tables[name]
	if !ok {
		return "", "", nil, false
	}
	return t.hashKey, t.rangeKey, t.gsis, true
}

// nameOnlyDynamo implements GetTableNames but NOT TableKeySchema, exercising
// the backward-compatible name-only comparison path.
type nameOnlyDynamo struct{ names []string }

func (n *nameOnlyDynamo) Name() string              { return "dynamodb" }
func (n *nameOnlyDynamo) Actions() []service.Action { return nil }
func (n *nameOnlyDynamo) HealthCheck() error        { return nil }
func (n *nameOnlyDynamo) HandleRequest(*service.RequestContext) (*service.Response, error) {
	return nil, nil
}
func (n *nameOnlyDynamo) GetTableNames() []string { return n.names }

// fakeRegistry resolves only the dynamodb service; everything else is absent.
type fakeRegistry struct{ dynamo service.Service }

func (r fakeRegistry) Lookup(name string) (service.Service, error) {
	if name == "dynamodb" && r.dynamo != nil {
		return r.dynamo, nil
	}
	return nil, errors.New("service not registered")
}

// entryFor returns the diff entry for a named table, or a zero entry.
func entryFor(res *DiffResult, name string) DiffEntry {
	for _, e := range res.Entries {
		if e.Name == name {
			return e
		}
	}
	return DiffEntry{}
}

// --- tests ----------------------------------------------------------------

func TestComputeDiff_DynamoDrift(t *testing.T) {
	// Running table: pk/sk with a "by-email" GSI on email.
	running := &fakeDynamo{tables: map[string]fakeTable{
		"users": {
			hashKey:  "pk",
			rangeKey: "sk",
			gsis:     map[string][2]string{"by-email": {"email", ""}},
		},
	}}

	tests := []struct {
		name        string
		decl        DynamoTableDef
		wantStatus  DiffStatus
		wantDetails []string // substrings expected in Details (drift only)
	}{
		{
			name:       "in sync",
			decl:       DynamoTableDef{Name: "users", HashKey: "pk", RangeKey: "sk", GSIs: []GSIDef{{Name: "by-email", HashKey: "email"}}},
			wantStatus: DiffSynced,
		},
		{
			name:        "hash key drift",
			decl:        DynamoTableDef{Name: "users", HashKey: "id", RangeKey: "sk", GSIs: []GSIDef{{Name: "by-email", HashKey: "email"}}},
			wantStatus:  DiffDrift,
			wantDetails: []string{`hash key "id" in IaC, "pk" running`},
		},
		{
			name:        "range key removed in IaC",
			decl:        DynamoTableDef{Name: "users", HashKey: "pk", GSIs: []GSIDef{{Name: "by-email", HashKey: "email"}}},
			wantStatus:  DiffDrift,
			wantDetails: []string{`range key "sk" running but absent in IaC`},
		},
		{
			name:        "gsi missing at runtime",
			decl:        DynamoTableDef{Name: "users", HashKey: "pk", RangeKey: "sk", GSIs: []GSIDef{{Name: "by-email", HashKey: "email"}, {Name: "by-status", HashKey: "status"}}},
			wantStatus:  DiffDrift,
			wantDetails: []string{`GSI "by-status" declared in IaC but not running`},
		},
		{
			name:        "gsi key mismatch",
			decl:        DynamoTableDef{Name: "users", HashKey: "pk", RangeKey: "sk", GSIs: []GSIDef{{Name: "by-email", HashKey: "emailAddr"}}},
			wantStatus:  DiffDrift,
			wantDetails: []string{`GSI "by-email" hash key "emailAddr" in IaC, "email" running`},
		},
		{
			name:        "extra gsi at runtime",
			decl:        DynamoTableDef{Name: "users", HashKey: "pk", RangeKey: "sk"},
			wantStatus:  DiffDrift,
			wantDetails: []string{`GSI "by-email" running but not declared in IaC`},
		},
		{
			name:       "empty hash key in IaC does not flag drift",
			decl:       DynamoTableDef{Name: "users", RangeKey: "sk", GSIs: []GSIDef{{Name: "by-email", HashKey: "email"}}},
			wantStatus: DiffSynced,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := ComputeDiff(&IaCImportResult{Tables: []DynamoTableDef{tc.decl}}, fakeRegistry{dynamo: running}, nil)
			got := entryFor(res, "users")
			if got.Status != tc.wantStatus {
				t.Fatalf("status = %q, want %q (details: %q)", got.Status, tc.wantStatus, got.Details)
			}
			for _, want := range tc.wantDetails {
				if !strings.Contains(got.Details, want) {
					t.Errorf("details %q missing substring %q", got.Details, want)
				}
			}
			if tc.wantStatus == DiffSynced && got.Details != "" {
				t.Errorf("synced entry should have no details, got %q", got.Details)
			}
		})
	}
}

func TestComputeDiff_DynamoMissingAndOrphaned(t *testing.T) {
	running := &fakeDynamo{tables: map[string]fakeTable{
		"orphan": {hashKey: "pk"},
	}}
	res := ComputeDiff(&IaCImportResult{Tables: []DynamoTableDef{
		{Name: "declared-not-running", HashKey: "pk"},
	}}, fakeRegistry{dynamo: running}, nil)

	if got := entryFor(res, "declared-not-running"); got.Status != DiffMissing {
		t.Errorf("declared-not-running: status = %q, want missing", got.Status)
	}
	if got := entryFor(res, "orphan"); got.Status != DiffOrphaned {
		t.Errorf("orphan: status = %q, want orphaned", got.Status)
	}
	if res.Summary.Missing != 1 || res.Summary.Orphaned != 1 {
		t.Errorf("summary = %+v, want Missing=1 Orphaned=1", res.Summary)
	}
}

// A service exposing only GetTableNames must still produce a name-only Synced
// result (no drift detection), preserving backward compatibility.
func TestComputeDiff_DynamoNoInspectorFallsBackToSynced(t *testing.T) {
	res := ComputeDiff(&IaCImportResult{Tables: []DynamoTableDef{
		{Name: "users", HashKey: "different", GSIs: []GSIDef{{Name: "x", HashKey: "y"}}},
	}}, fakeRegistry{dynamo: &nameOnlyDynamo{names: []string{"users"}}}, nil)

	if got := entryFor(res, "users"); got.Status != DiffSynced {
		t.Errorf("status = %q, want synced (no inspector → name-only match)", got.Status)
	}
}
