package report_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	incmemory "github.com/neureaux/cloudmock/pkg/incident/memory"
	regmemory "github.com/neureaux/cloudmock/pkg/regression/memory"

	"github.com/neureaux/cloudmock/pkg/incident"
	"github.com/neureaux/cloudmock/pkg/regression"
	"github.com/neureaux/cloudmock/pkg/report"
)

func newTestIncident(id string) *incident.Incident {
	now := time.Now().UTC()
	return &incident.Incident{
		ID:               id,
		Status:           "active",
		Severity:         "critical",
		Title:            "High error rate on api-gateway",
		AffectedServices: []string{"api-gateway", "auth-service"},
		FirstSeen:        now.Add(-30 * time.Minute),
		LastSeen:         now,
	}
}

func newTestRegression(svc string) *regression.Regression {
	return &regression.Regression{
		Algorithm:     regression.AlgoErrorRate,
		Severity:      regression.SeverityCritical,
		Confidence:    92,
		Service:       svc,
		Title:         "Error rate spike",
		ChangePercent: 150.0,
		DetectedAt:    time.Now().UTC().Add(-25 * time.Minute),
		Status:        "active",
	}
}

func seedStores(t *testing.T) (incident.IncidentStore, regression.RegressionStore, *incident.Incident, *regression.Regression) {
	t.Helper()
	ctx := context.Background()

	incStore := incmemory.NewStore()
	regStore := regmemory.NewStore()

	inc := newTestIncident("")
	if err := incStore.Save(ctx, inc); err != nil {
		t.Fatalf("saving incident: %v", err)
	}

	reg := newTestRegression("api-gateway")
	if err := regStore.Save(ctx, reg); err != nil {
		t.Fatalf("saving regression: %v", err)
	}

	return incStore, regStore, inc, reg
}

func TestGenerate_JSON(t *testing.T) {
	incStore, regStore, inc, _ := seedStores(t)
	gen := report.New(incStore, regStore, nil)

	data, ct, err := gen.Generate(context.Background(), inc.ID, "json")
	if err != nil {
		t.Fatalf("Generate JSON: %v", err)
	}
	if !strings.Contains(ct, "application/json") {
		t.Errorf("content-type = %q, want application/json", ct)
	}

	var rpt report.Report
	if err := json.Unmarshal(data, &rpt); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}
	if rpt.Incident.ID != inc.ID {
		t.Errorf("incident ID = %q, want %q", rpt.Incident.ID, inc.ID)
	}
	if rpt.Incident.Title != inc.Title {
		t.Errorf("incident title = %q, want %q", rpt.Incident.Title, inc.Title)
	}
	if len(rpt.Regressions) == 0 {
		t.Error("expected at least one regression in report")
	}
	if len(rpt.Timeline) == 0 {
		t.Error("expected non-empty timeline")
	}
	if rpt.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should not be zero")
	}
}

func TestGenerate_CSV(t *testing.T) {
	incStore, regStore, inc, _ := seedStores(t)
	gen := report.New(incStore, regStore, nil)

	data, ct, err := gen.Generate(context.Background(), inc.ID, "csv")
	if err != nil {
		t.Fatalf("Generate CSV: %v", err)
	}
	if !strings.Contains(ct, "text/csv") {
		t.Errorf("content-type = %q, want text/csv", ct)
	}

	body := string(data)
	for _, want := range []string{
		"Section,Field,Value",
		inc.ID,
		inc.Title,
		"critical",
		"active",
		"api-gateway",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("CSV output missing %q", want)
		}
	}
}

func TestGenerate_HTML(t *testing.T) {
	incStore, regStore, inc, _ := seedStores(t)
	gen := report.New(incStore, regStore, nil)

	data, ct, err := gen.Generate(context.Background(), inc.ID, "html")
	if err != nil {
		t.Fatalf("Generate HTML: %v", err)
	}
	if !strings.Contains(ct, "text/html") {
		t.Errorf("content-type = %q, want text/html", ct)
	}

	body := string(data)
	for _, want := range []string{
		"<!DOCTYPE html>",
		inc.Title,
		"severity-critical",
		"Incident Report:",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("HTML output missing %q", want)
		}
	}
}

func TestGenerate_NotFound(t *testing.T) {
	incStore := incmemory.NewStore()
	gen := report.New(incStore, nil, nil)

	_, _, err := gen.Generate(context.Background(), "nonexistent-id", "json")
	if err == nil {
		t.Fatal("expected error for nonexistent incident, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want message containing 'not found'", err.Error())
	}
}

func TestGenerate_InvalidFormat(t *testing.T) {
	incStore, regStore, inc, _ := seedStores(t)
	gen := report.New(incStore, regStore, nil)

	_, _, err := gen.Generate(context.Background(), inc.ID, "xml")
	if err == nil {
		t.Fatal("expected error for unsupported format, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("error = %q, want message containing 'unsupported format'", err.Error())
	}
}
