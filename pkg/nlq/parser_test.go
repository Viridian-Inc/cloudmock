package nlq

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		question    string
		wantType    string
		wantService string
		wantErr     bool
		wantFilter  string // optional: check a specific filter key
		wantFilterV string // optional: check a specific filter value
	}{
		{
			question:    "what's the error rate for attendance?",
			wantType:    "errors",
			wantService: "attendance",
		},
		{
			question: "show me slow requests",
			wantType: "requests",
			wantFilter: "latency",
			wantFilterV: "p95",
		},
		{
			question: "which service has the most errors?",
			wantType: "errors",
			wantFilter: "aggregate",
			wantFilterV: "service",
		},
		{
			question: "what happened after the last deploy?",
			wantType: "errors",
			wantFilter: "since",
			wantFilterV: "last_deploy",
		},
		{
			question: "how is the system doing?",
			wantType: "health",
		},
		{
			question:    "show me slow requests for billing",
			wantType:    "requests",
			wantService: "billing",
		},
		{
			question: "show me the latest deployments",
			wantType: "deploys",
		},
		{
			question: "what are the cpu metrics?",
			wantType: "metrics",
		},
		{
			question:    "errors in auth service last hour",
			wantType:    "errors",
			wantService: "auth",
		},
		{
			question: "overall status",
			wantType: "health",
		},
		{
			question:    "request traffic for payments last 24h",
			wantType:    "requests",
			wantService: "payments",
		},
		{
			question:    "what are the p99 metrics for gateway",
			wantType:    "metrics",
			wantService: "gateway",
		},
		{
			question: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.question, func(t *testing.T) {
			query, err := Parse(tt.question)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if query.Type != tt.wantType {
				t.Errorf("type = %q, want %q", query.Type, tt.wantType)
			}
			if tt.wantService != "" && query.Service != tt.wantService {
				t.Errorf("service = %q, want %q", query.Service, tt.wantService)
			}
			if tt.wantFilter != "" {
				v, ok := query.Filters[tt.wantFilter]
				if !ok {
					t.Errorf("missing filter %q", tt.wantFilter)
				} else if v != tt.wantFilterV {
					t.Errorf("filter[%q] = %q, want %q", tt.wantFilter, v, tt.wantFilterV)
				}
			}
		})
	}
}

func TestDescribe(t *testing.T) {
	q := &Query{
		Type:    "errors",
		Service: "attendance",
		Filters: map[string]string{},
	}
	desc := q.Describe()
	if desc == "" {
		t.Fatal("Describe returned empty string")
	}
	if !contains(desc, "error") {
		t.Errorf("description %q should mention errors", desc)
	}
	if !contains(desc, "attendance") {
		t.Errorf("description %q should mention service name", desc)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
