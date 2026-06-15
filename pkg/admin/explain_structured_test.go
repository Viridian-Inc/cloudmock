package admin

import (
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/gateway"
)

func TestBuildExplainStructured_AffectedServices(t *testing.T) {
	entry := &gateway.RequestEntry{Service: "dynamodb", Action: "Query", StatusCode: 200, LatencyMs: 5}
	ctx := &ExplainContext{Timeline: []gateway.TimelineSpan{{Service: "bff"}, {Service: "dynamodb"}}}
	s := buildExplainStructured(entry, ctx, &ExplainAnalysis{}, nil)

	// request service first, timeline services added, deduped.
	if len(s.AffectedServices) != 2 || s.AffectedServices[0] != "dynamodb" || s.AffectedServices[1] != "bff" {
		t.Errorf("AffectedServices = %v, want [dynamodb bff]", s.AffectedServices)
	}
}

func TestBuildExplainStructured_ErrorWithDeploy(t *testing.T) {
	entry := &gateway.RequestEntry{Service: "dynamodb", Action: "PutItem", StatusCode: 500, LatencyMs: 12}
	deploys := []RelatedDeploy{{Service: "bff", Author: "alice", Commit: "abcdef1234567890", Message: "ship"}}
	s := buildExplainStructured(entry, &ExplainContext{}, &ExplainAnalysis{IsError: true, ErrorRate: 0.05}, deploys)

	if !strings.Contains(s.ProbableCause, "recent deploy of bff") {
		t.Errorf("ProbableCause = %q, want deploy correlation", s.ProbableCause)
	}
	if !strings.Contains(s.ProbableCause, "abcdef12") { // commit truncated to 8
		t.Errorf("ProbableCause should use short commit: %q", s.ProbableCause)
	}
	if !strings.Contains(s.SuggestedFix, "roll back") {
		t.Errorf("SuggestedFix = %q, want rollback advice", s.SuggestedFix)
	}
	if len(s.RelatedDeploys) != 1 {
		t.Errorf("RelatedDeploys = %d, want 1", len(s.RelatedDeploys))
	}
}

func TestBuildExplainStructured_SlowAndNormal(t *testing.T) {
	// Slow with a slowest span.
	entry := &gateway.RequestEntry{Service: "api", Action: "Get", StatusCode: 200, LatencyMs: 400}
	a := &ExplainAnalysis{IsSlow: true, LatencyRatio: 4, P50Ms: 100, SlowestSpan: "dynamodb/Scan"}
	s := buildExplainStructured(entry, &ExplainContext{}, a, nil)
	if !strings.Contains(s.ProbableCause, "dynamodb/Scan") || !strings.Contains(s.SuggestedFix, "Optimize dynamodb/Scan") {
		t.Errorf("slow case = %+v", s)
	}

	// Normal request → benign cause, no fix.
	s = buildExplainStructured(
		&gateway.RequestEntry{Service: "s3", Action: "GetObject", StatusCode: 200, LatencyMs: 8},
		&ExplainContext{}, &ExplainAnalysis{}, nil)
	if !strings.Contains(s.ProbableCause, "completed normally") {
		t.Errorf("normal ProbableCause = %q", s.ProbableCause)
	}
	if s.SuggestedFix != "" {
		t.Errorf("normal request should have no SuggestedFix, got %q", s.SuggestedFix)
	}
}
